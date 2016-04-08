package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/imports"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/nomdl/codegen/code"
	"github.com/attic-labs/noms/nomdl/pkg"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

const goExt = "go"
const jsExt = "js"

var (
	outDirFlag  = flag.String("out-dir", ".", "Directory where generated code will be written")
	inFlag      = flag.String("in", "", "The name of the noms file to read")
	pkgDSFlag   = flag.String("package-ds", "", "The dataset to read/write packages from/to.")
	packageFlag = flag.String("package", "", "The name of the go package to write")
	outputLang  = flag.String("out-lang", goExt, `Output language. Supported values are "go" and "js"`)

	idRegexp    = regexp.MustCompile(`[_\pL][_\pL\pN]*`)
	illegalRune = regexp.MustCompile(`[^_\pL\pN]`)
)

const ext = ".noms"

type refSet map[ref.Ref]bool

func main() {
	flags := datas.NewFlags()
	flag.Parse()

	ds, ok := flags.CreateDataStore()
	if !ok {
		ds = datas.NewDataStore(chunks.NewMemoryStore())
	}
	defer ds.Close()

	if *pkgDSFlag != "" {
		if !ok {
			log.Print("Package dataset provided, but DataStore could not be opened.")
			flag.Usage()
			return
		}
	} else {
		log.Print("No package dataset provided; will be unable to process imports.")
		*pkgDSFlag = "default"
	}

	if *outputLang != goExt && *outputLang != jsExt {
		log.Print("Invalid out-lang provided.")
		flag.Usage()
		return
	}

	pkgDS := dataset.NewDataset(ds, *pkgDSFlag)
	// Ensure that, if pkgDS has stuff in it, its head is a SetOfRefOfPackage.
	if h, ok := pkgDS.MaybeHead(); ok {
		d.Chk.IsType(types.SetOfRefOfPackage{}, h.Value())
	}

	localPkgs := refSet{}
	outDir, err := filepath.Abs(*outDirFlag)
	d.Chk.NoError(err, "Could not canonicalize -out-dir: %v", err)
	packageName := ""
	if *outputLang == goExt {
		packageName = getGoPackageName(outDir)
	}

	if *inFlag != "" {
		out := getOutFileName(filepath.Base(*inFlag))
		p := parsePackageFile(packageName, *inFlag, pkgDS)
		localPkgs[p.Ref()] = true
		generate(packageName, *inFlag, filepath.Join(outDir, out), outDir, map[string]bool{}, p, localPkgs, pkgDS)
		return
	}

	// Generate code from all .noms file in the current directory
	nomsFiles, err := filepath.Glob("*" + ext)
	d.Chk.NoError(err)

	written := map[string]bool{}
	packages := map[string]pkg.Parsed{}
	for _, inFile := range nomsFiles {
		p := parsePackageFile(packageName, inFile, pkgDS)
		localPkgs[p.Ref()] = true
		packages[inFile] = p
	}
	// Sort to have deterministic output.
	keys := make([]string, 0, len(packages))
	sort.Strings(keys)
	for inFile := range packages {
		keys = append(keys, inFile)
	}
	for _, inFile := range keys {
		p := packages[inFile]
		pkgDS = generate(packageName, inFile, filepath.Join(outDir, getOutFileName(inFile)), outDir, written, p, localPkgs, pkgDS)
	}
}

func parsePackageFile(packageName string, in string, pkgDS dataset.Dataset) pkg.Parsed {
	inFile, err := os.Open(in)
	d.Chk.NoError(err)
	defer inFile.Close()

	return pkg.ParseNomDL(packageName, inFile, filepath.Dir(in), pkgDS.Store())
}

func generate(packageName, in, out, outDir string, written map[string]bool, parsed pkg.Parsed, localPkgs refSet, pkgDS dataset.Dataset) dataset.Dataset {
	// Generate code for all p's deps first.
	deps := generateDepCode(packageName, outDir, written, parsed.Package, localPkgs, pkgDS.Store())
	generateAndEmit(getBareFileName(in), out, written, deps, parsed)

	// Since we're just building up a set of refs to all the packages in pkgDS, simply retrying is the logical response to commit failure.
	err := datas.ErrOptimisticLockFailed
	for ; err == datas.ErrOptimisticLockFailed; pkgDS, err = pkgDS.Commit(buildSetOfRefOfPackage(parsed, deps, pkgDS)) {
	}
	return pkgDS
}

type depsMap map[ref.Ref]types.Package

func generateDepCode(packageName, outDir string, written map[string]bool, p types.Package, localPkgs refSet, vr types.ValueReader) depsMap {
	deps := depsMap{}
	for _, r := range p.Dependencies() {
		p := vr.ReadValue(r).(types.Package)
		pDeps := generateDepCode(packageName, outDir, written, p, localPkgs, vr)
		tag := code.ToTag(p.Ref())
		parsed := pkg.Parsed{Package: p, Name: packageName}
		if !localPkgs[parsed.Ref()] {
			generateAndEmit(tag, filepath.Join(outDir, tag+"."+*outputLang), written, pDeps, parsed)
			localPkgs[parsed.Ref()] = true
		}
		for depRef, dep := range pDeps {
			deps[depRef] = dep
		}
		deps[r] = p
	}
	return deps
}

func generateAndEmit(tag, out string, written map[string]bool, deps depsMap, p pkg.Parsed) {
	var buf bytes.Buffer
	gen := newCodeGen(&buf, tag, *outputLang, written, deps, p)
	gen.WritePackage()

	if *outputLang == goExt {
		// go format
		bs, err := imports.Process(out, buf.Bytes(), nil)
		if err != nil {
			fmt.Println(buf.String())
		}
		d.Chk.NoError(err)
		buf = *bytes.NewBuffer(bs)
	}

	d.Chk.NoError(os.MkdirAll(filepath.Dir(out), 0700))

	outFile, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	d.Chk.NoError(err)
	defer outFile.Close()

	io.Copy(outFile, &buf)
}

func buildSetOfRefOfPackage(pkg pkg.Parsed, deps depsMap, ds dataset.Dataset) types.SetOfRefOfPackage {
	// Can do better once generated collections implement types.Value.
	s := types.NewSetOfRefOfPackage()
	if h, ok := ds.MaybeHead(); ok {
		s = h.Value().(types.SetOfRefOfPackage)
	}
	for _, dep := range deps {
		// Writing the deps into ds should be redundant at this point, but do it to be sure.
		// TODO: consider moving all dataset work over into nomdl/pkg BUG 409
		s = s.Insert(types.NewRefOfPackage(ds.Store().WriteValue(dep)))
	}
	r := ds.Store().WriteValue(pkg.Package)
	return s.Insert(types.NewRefOfPackage(r))
}

func getOutFileName(in string) string {
	return in[:len(in)-len(ext)] + ".noms." + *outputLang
}

func getBareFileName(in string) string {
	base := filepath.Base(in)
	return base[:len(base)-len(filepath.Ext(base))]
}

func getGoPackageName(outDir string) string {
	if *packageFlag != "" {
		d.Exp.True(idRegexp.MatchString(*packageFlag), "%s is not a legal Go identifier.", *packageFlag)
		return *packageFlag
	}

	// It is illegal to have multiple go files in the same directory with different package names.
	// We can therefore just pick the first one and get the package name from there.
	goFiles, err := filepath.Glob(filepath.Join(outDir, "*.go"))
	d.Chk.NoError(err)
	if len(goFiles) == 0 {
		d.Exp.NotEmpty(outDir, "Cannot convert empty path into a Go package name.")
		return makeGoIdentifier(filepath.Base(outDir))
	}
	d.Chk.True(len(goFiles) > 0, "No Go files in current directory; cannot infer pacakge name.")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, goFiles[0], nil, parser.PackageClauseOnly)
	d.Chk.NoError(err)
	return f.Name.String()
}

func makeGoIdentifier(in string) string {
	d.Chk.NotEmpty(in, "Cannot convert empty string to legal Go identifier.")
	if r, _ := utf8.DecodeRuneInString(in); unicode.IsNumber(r) {
		in = "_" + in
	}
	return illegalRune.ReplaceAllLiteralString(in, "_")
}

type codeGen struct {
	w          io.Writer
	pkg        pkg.Parsed
	deps       depsMap
	written    map[string]bool
	toWrite    []types.Type
	generator  *code.Generator
	lang       string
	templates  *template.Template
	sharedData sharedData
}

func newCodeGen(w io.Writer, fileID, lang string, written map[string]bool, deps depsMap, pkg pkg.Parsed) *codeGen {
	typesPackage := "types."
	if pkg.Name == "types" {
		typesPackage = ""
	}
	nomsImport := "github.com/attic-labs/noms"
	gen := &codeGen{w, pkg, deps, written, []types.Type{}, nil, lang, nil, sharedData{
		fileID,
		nomsImport,
		pkg.Name,
		typesPackage,
	}}
	gen.generator = &code.Generator{R: gen, TypesPackage: typesPackage, AliasNames: pkg.AliasNames}
	gen.templates = gen.readTemplates()
	return gen
}

func (gen *codeGen) readTemplates() *template.Template {
	_, thisfile, _, _ := runtime.Caller(1)
	glob := path.Join(path.Dir(thisfile), gen.lang, "*.tmpl")
	return template.Must(template.New("").Funcs(
		template.FuncMap{
			"defToUser":            gen.generator.DefToUser,
			"defToValue":           gen.generator.DefToValue,
			"defType":              gen.generator.DefType,
			"importJS":             gen.generator.ImportJS,
			"importJsType":         gen.generator.ImportJSType,
			"isLast":               gen.generator.IsLast,
			"mayHaveChunks":        gen.generator.MayHaveChunks,
			"refToAliasName":       gen.generator.RefToAliasName,
			"refToJSIdentfierName": gen.generator.RefToJSIdentfierName,
			"title":                strings.Title,
			"toTypesType":          gen.generator.ToType,
			"toTypeValueJS":        gen.generator.ToTypeValueJS,
			"userToDef":            gen.generator.UserToDef,
			"userToValue":          gen.generator.UserToValue,
			"userType":             gen.generator.UserType,
			"userTypeJS":           gen.generator.UserTypeJS,
			"userZero":             gen.generator.UserZero,
			"valueToDef":           gen.generator.ValueToDef,
			"valueToUser":          gen.generator.ValueToUser,
			"valueZero":            gen.generator.ValueZero,
		}).ParseGlob(glob))
}

func (gen *codeGen) Resolve(t types.Type) types.Type {
	if !t.IsUnresolved() {
		return t
	}
	if !t.HasPackageRef() {
		return gen.pkg.Types()[t.Ordinal()]
	}

	dep, ok := gen.deps[t.PackageRef()]
	d.Chk.True(ok, "Package %s is referenced in %+v, but is not a dependency.", t.PackageRef().String(), t)
	return dep.Types()[t.Ordinal()]
}

type sharedData struct {
	FileID       string
	NomsImport   string
	PackageName  string
	TypesPackage string
}

func (gen *codeGen) WritePackage() {
	pkgTypes := gen.pkg.Types()
	data := struct {
		sharedData
		HasTypes     bool
		Dependencies []ref.Ref
		Name         string
		Types        []types.Type
	}{
		gen.sharedData,
		len(pkgTypes) > 0,
		gen.pkg.Dependencies(),
		gen.pkg.Name,
		pkgTypes,
	}

	// In JS we want to write the imports at the top of the file but we do not know what we need to import until we have written everything. We therefore write to a buffer and when everything is done we can write the imports and write the buffer into the writer.
	var buf bytes.Buffer
	w := gen.w

	if gen.lang == jsExt {
		gen.w = &buf
	} else {
		gen.WriteHeader()
	}

	err := gen.templates.ExecuteTemplate(gen.w, "package.tmpl", data)
	d.Exp.NoError(err)

	for i, t := range pkgTypes {
		gen.writeTopLevel(t, i)
	}

	for _, t := range gen.pkg.UsingDeclarations {
		gen.write(t)
	}

	for len(gen.toWrite) > 0 {
		t := gen.toWrite[0]
		gen.toWrite = gen.toWrite[1:]
		gen.write(t)
	}

	if gen.lang == jsExt {
		gen.w = w
		gen.WriteHeader()
		io.Copy(w, &buf)
	}
}

func (gen *codeGen) WriteHeader() {
	importedJS := make([]string, 0, len(gen.generator.ImportedJS))
	importedJSTypes := make([]string, 0, len(gen.generator.ImportedJSTypes))
	if gen.lang == jsExt {
		for name := range gen.generator.ImportedJS {
			importedJS = append(importedJS, name)
		}
		for name := range gen.generator.ImportedJSTypes {
			if _, ok := gen.generator.ImportedJS[name]; !ok {
				importedJSTypes = append(importedJSTypes, name)
			}
		}
		sort.Strings(importedJS)
		sort.Strings(importedJSTypes)
	}

	pkgTypes := gen.pkg.Types()
	data := struct {
		sharedData
		HasTypes        bool
		Dependencies    []ref.Ref
		Name            string
		Types           []types.Type
		ImportedJS      []string
		ImportedJSTypes []string
		AliasNames      map[ref.Ref]string
	}{
		gen.sharedData,
		len(pkgTypes) > 0,
		gen.pkg.Dependencies(),
		gen.pkg.Name,
		pkgTypes,
		importedJS,
		importedJSTypes,
		gen.pkg.AliasNames,
	}

	err := gen.templates.ExecuteTemplate(gen.w, "header.tmpl", data)
	d.Exp.NoError(err)
}

func (gen *codeGen) shouldBeWritten(t types.Type) bool {
	if t.IsUnresolved() {
		return false
	}
	if t.Kind() == types.EnumKind || t.Kind() == types.StructKind {
		name := gen.generator.UserName(t)
		d.Chk.False(gen.written[name], "Multiple definitions of type named %s", name)
		return true
	}
	return !gen.written[gen.generator.UserName(t)]
}

func (gen *codeGen) writeTopLevel(t types.Type, ordinal int) {
	switch t.Kind() {
	case types.EnumKind:
		gen.writeEnum(t, ordinal)
	case types.StructKind:
		gen.writeStruct(t, ordinal)
	default:
		gen.write(t)
	}
}

// write generates the code for the given type.
func (gen *codeGen) write(t types.Type) {
	if !gen.shouldBeWritten(t) {
		return
	}
	k := t.Kind()
	switch k {
	case types.BlobKind, types.BoolKind, types.Float32Kind, types.Float64Kind, types.Int16Kind, types.Int32Kind, types.Int64Kind, types.Int8Kind, types.PackageKind, types.StringKind, types.Uint16Kind, types.Uint32Kind, types.Uint64Kind, types.Uint8Kind, types.ValueKind, types.TypeKind:
		return
	case types.ListKind:
		gen.writeList(t)
	case types.MapKind:
		gen.writeMap(t)
	case types.RefKind:
		gen.writeRef(t)
	case types.SetKind:
		gen.writeSet(t)
	default:
		panic("unreachable")
	}
}

func (gen *codeGen) writeLater(t types.Type) {
	if !gen.shouldBeWritten(t) {
		return
	}
	gen.toWrite = append(gen.toWrite, t)
}

func (gen *codeGen) writeTemplate(tmpl string, t types.Type, data interface{}) {
	err := gen.templates.ExecuteTemplate(gen.w, tmpl, data)
	d.Exp.NoError(err)
	gen.written[gen.generator.UserName(t)] = true
}

func (gen *codeGen) writeStruct(t types.Type, ordinal int) {
	d.Chk.True(ordinal >= 0)
	desc := t.Desc.(types.StructDesc)
	data := struct {
		sharedData
		Name          string
		Type          types.Type
		Ordinal       int
		Fields        []types.Field
		Choices       types.Choices
		HasUnion      bool
		UnionZeroType types.Type
		CanUseDef     bool
	}{
		gen.sharedData,
		gen.generator.UserName(t),
		t,
		ordinal,
		desc.Fields,
		nil,
		len(desc.Union) != 0,
		types.MakePrimitiveType(types.Uint32Kind),
		gen.canUseDef(t, gen.pkg.Package),
	}

	if data.HasUnion {
		data.Choices = desc.Union
		data.UnionZeroType = data.Choices[0].T
	}
	gen.writeTemplate("struct.tmpl", t, data)
	for _, f := range desc.Fields {
		gen.writeLater(f.T)
	}
	if data.HasUnion {
		for _, f := range desc.Union {
			gen.writeLater(f.T)
		}
	}
}

func (gen *codeGen) writeList(t types.Type) {
	elemTypes := t.Desc.(types.CompoundDesc).ElemTypes
	data := struct {
		sharedData
		Name      string
		Type      types.Type
		ElemType  types.Type
		CanUseDef bool
	}{
		gen.sharedData,
		gen.generator.UserName(t),
		t,
		elemTypes[0],
		gen.canUseDef(t, gen.pkg.Package),
	}
	gen.writeTemplate("list.tmpl", t, data)
	gen.writeLater(elemTypes[0])
}

func (gen *codeGen) writeMap(t types.Type) {
	elemTypes := t.Desc.(types.CompoundDesc).ElemTypes
	data := struct {
		sharedData
		Name      string
		Type      types.Type
		KeyType   types.Type
		ValueType types.Type
		CanUseDef bool
	}{
		gen.sharedData,
		gen.generator.UserName(t),
		t,
		elemTypes[0],
		elemTypes[1],
		gen.canUseDef(t, gen.pkg.Package),
	}
	gen.writeTemplate("map.tmpl", t, data)
	gen.writeLater(elemTypes[0])
	gen.writeLater(elemTypes[1])
}

func (gen *codeGen) writeRef(t types.Type) {
	elemTypes := t.Desc.(types.CompoundDesc).ElemTypes
	data := struct {
		sharedData
		Name     string
		Type     types.Type
		ElemType types.Type
	}{
		gen.sharedData,
		gen.generator.UserName(t),
		t,
		elemTypes[0],
	}
	gen.writeTemplate("ref.tmpl", t, data)
	gen.writeLater(elemTypes[0])
}

func (gen *codeGen) writeSet(t types.Type) {
	elemTypes := t.Desc.(types.CompoundDesc).ElemTypes
	data := struct {
		sharedData
		Name      string
		Type      types.Type
		ElemType  types.Type
		CanUseDef bool
	}{
		gen.sharedData,
		gen.generator.UserName(t),
		t,
		elemTypes[0],
		gen.canUseDef(t, gen.pkg.Package),
	}
	gen.writeTemplate("set.tmpl", t, data)
	gen.writeLater(elemTypes[0])
}

func (gen *codeGen) writeEnum(t types.Type, ordinal int) {
	d.Chk.True(ordinal >= 0)
	data := struct {
		sharedData
		Name    string
		Type    types.Type
		Ordinal int
		Ids     []string
	}{
		gen.sharedData,
		t.Name(),
		t,
		ordinal,
		t.Desc.(types.EnumDesc).IDs,
	}

	gen.writeTemplate("enum.tmpl", t, data)
}

func (gen *codeGen) canUseDef(t types.Type, p types.Package) bool {
	cache := map[string]bool{}

	var rec func(t types.Type, p types.Package) bool
	rec = func(t types.Type, p types.Package) bool {
		switch t.Kind() {
		case types.UnresolvedKind:
			t2, p2 := gen.resolveInPackage(t, p)
			d.Chk.False(t2.IsUnresolved())
			return rec(t2, p2)
		case types.ListKind:
			return rec(t.Desc.(types.CompoundDesc).ElemTypes[0], p)
		case types.SetKind:
			elemType := t.Desc.(types.CompoundDesc).ElemTypes[0]
			return !gen.containsNonComparable(elemType, p) && rec(elemType, p)
		case types.MapKind:
			elemTypes := t.Desc.(types.CompoundDesc).ElemTypes
			return !gen.containsNonComparable(elemTypes[0], p) && rec(elemTypes[0], p) && rec(elemTypes[1], p)
		case types.StructKind:
			userName := gen.generator.UserName(t)
			if b, ok := cache[userName]; ok {
				return b
			}
			cache[userName] = true
			for _, f := range t.Desc.(types.StructDesc).Fields {
				if f.T.Equals(t) || !rec(f.T, p) {
					cache[userName] = false
					return false
				}
			}
			return true
		default:
			return true
		}
	}

	return rec(t, p)
}

// We use a go map as the def for Set and Map. These cannot have a key that is a
// Set, Map or a List because slices and maps are not comparable in go.
func (gen *codeGen) containsNonComparable(t types.Type, p types.Package) bool {
	cache := map[string]bool{}

	var rec func(t types.Type, p types.Package) bool
	rec = func(t types.Type, p types.Package) bool {
		switch t.Desc.Kind() {
		case types.UnresolvedKind:
			t2, p2 := gen.resolveInPackage(t, p)
			d.Chk.False(t2.IsUnresolved())
			return rec(t2, p2)
		case types.ListKind, types.MapKind, types.SetKind:
			return true
		case types.StructKind:
			// Only structs can be recursive
			userName := gen.generator.UserName(t)
			if b, ok := cache[userName]; ok {
				return b
			}
			// If we get here in a recursive call we will mark it as not having a non comparable value. If it does then that will get handled higher up in the call chain.
			cache[userName] = false
			for _, f := range t.Desc.(types.StructDesc).Fields {
				if rec(f.T, p) {
					cache[userName] = true
					return true
				}
			}
			return cache[userName]
		default:
			return false
		}
	}

	return rec(t, p)
}

func (gen *codeGen) resolveInPackage(t types.Type, p types.Package) (types.Type, types.Package) {
	d.Chk.True(t.IsUnresolved())

	// For unresolved types that references types in the same package the ref is empty and we need to use the passed in package.
	if t.HasPackageRef() {
		p = gen.deps[t.PackageRef()]
		d.Chk.NotNil(p)
	}

	return p.Types()[t.Ordinal()], p
}
