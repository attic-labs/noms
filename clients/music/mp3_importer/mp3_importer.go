package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
	id3 "github.com/mikkyang/id3-go"
)

func printSong(song Song) {
	fmt.Println("     Title:", song.Title())
	fmt.Println("    Artist:", song.Artist())
	fmt.Println("     Album:", song.Album())
	fmt.Println("      Year:", song.Year())
	fmt.Println("      Size:", song.Mp3().Len())
}

// Slurps mp3 files into a noms db.
//
// Depends on github.com/mikkyang/id3-go.
// To install, clone https://github.com/mikkyang/id3-go.git to the
// github.com/mikkyang directory alongside attic-labs.
//
// Possible usage, if you have mp3 files in your Music directory:
//
// find ~/Music -name '*.mp3' -exec ./mp3_importer -ldb /tmp/mp3_importer -ds main -mp3 {} \;

func main() {
	//
	// Set up noms.
	//

	dsFlags := dataset.NewFlags()
	mp3_flag := flag.String("mp3", "in.mp3", "Path to mp3 to import")
	flag.Parse()

	ds := dsFlags.CreateDataset()
	if ds == nil {
		flag.Usage()
		return
	}
	defer ds.Close()

	//
	// Read mp3.
	//

	mp3_filename := *mp3_flag

	// id3 data.
	fmt.Println("Reading id3 data")

	id3_data, err := id3.Open(mp3_filename)
	if err != nil {
		fmt.Println("Failed to read id3 data", mp3_filename, "with", err)
		return
	}
	defer id3_data.Close()

	// Song data (straight into noms).
	fmt.Println("Reading song data")

	mp3_file, err := os.Open(mp3_filename)
	if err != nil {
		fmt.Println("Failed to open", mp3_filename, "with", err)
		return
	}
	defer mp3_file.Close()

	fmt.Println("Convering to noms blob")
	mp3_data, err := types.NewBlob(bufio.NewReader(mp3_file))
	if err != nil {
		fmt.Println("Failed to read mp3 data", mp3_filename, "with", err)
		return
	}

	//
	// Read existing mp3 data.
	//

	fmt.Println("Reading existing data")

	songs := NewMapOfStringToSong()
	if commit, ok := ds.MaybeHead(); ok {
		songs = MapOfStringToSongFromVal(commit.Value())
	}

	fmt.Println("There are", songs.Len(), "existing songs:")
	songs.IterAll(func(k string, song Song) {
		fmt.Println("  Found", k)
		printSong(song)
	})

	//
	// Write new data.
	//

	new_song_key := fmt.Sprintf("%s.%s.%s.%s",
		id3_data.Title(), id3_data.Artist(), id3_data.Album(), id3_data.Year())
	new_song := NewSong()
	new_song = new_song.SetTitle(id3_data.Title())
	new_song = new_song.SetArtist(id3_data.Artist())
	new_song = new_song.SetAlbum(id3_data.Album())
	new_song = new_song.SetYear(id3_data.Year())
	new_song = new_song.SetMp3(mp3_data)
	songs = songs.Set(new_song_key, new_song)

	if _, ok := ds.Commit(songs.NomsValue()); !ok {
		fmt.Println("Failed to commit")
		return
	}

	fmt.Println("Committed:")
	printSong(new_song)
}
