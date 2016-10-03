//
//  NomsLocal.swift
//  NomsUI
//
// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0
//

import Foundation
import Noms4ios


let demoNomsIoServer = "https://demo.noms.io/cli-tour"

let developerServerKey = "developerServerKey"

func downloadDirectory() -> URL {
    let downloadDirURL = NSURL(fileURLWithPath: NSTemporaryDirectory()).appendingPathComponent("downloads")
    do {
        try FileManager.default.createDirectory(at: downloadDirURL!, withIntermediateDirectories: true, attributes: nil)
    } catch {
        // want to return the URL even if createDirectory fails
    }

    return downloadDirURL!
}


func uploadDirectory() -> URL {
    let uploadDirURL = NSURL(fileURLWithPath: NSTemporaryDirectory()).appendingPathComponent("uploads")
    do {
        try FileManager.default.createDirectory(at: uploadDirURL!, withIntermediateDirectories: true, attributes: nil)
    } catch {
        // want to return the URL even if createDirectory fails
    }

    return uploadDirURL!
}

func deleteFile(_ url: URL) {
    do {
        try FileManager.default.removeItem(at: url)
    } catch {
        print("unable to delete file: \(url)")
    }
}

func localDatasetsDirectory() -> URL {
    let documentsDirectory = FileManager().urls(for: .documentDirectory, in: .userDomainMask).first!
    let datasetDirURL = documentsDirectory.appendingPathComponent("datasets")

    do {
        try FileManager.default.createDirectory(at: datasetDirURL, withIntermediateDirectories: true, attributes: nil)
    } catch {
        // want to return the URL even if createDirectory fails
        print("unable to create dir: \(datasetDirURL)")
    }

    return datasetDirURL
}

func stripFileUrlScheme(_ path: URL) -> String {
    let str = path.absoluteString
    let range = str.index(str.startIndex, offsetBy: 7)..<str.endIndex
    let newPath = str.substring(with: range)
    return newPath
}

var developerNomsServerAddress: String? {
    get {
        let userDefaults = UserDefaults.standard
        return userDefaults.string(forKey: developerServerKey)
    }
    set {
        let userDefaults = UserDefaults.standard
        if newValue == nil {
            userDefaults.removeObject(forKey: developerServerKey)
        } else {
            userDefaults.set(newValue!, forKey: developerServerKey)
        }
    }
}

func datasetsAvailable(_ db: String) -> [String] {
    let datasetNamesString = Noms4ios.GoNoms4iosDsList(db)!
    if datasetNamesString == "" {
        return []
    } else {
        let datasetNamesArr = datasetNamesString.components(separatedBy: ",")
        return datasetNamesArr
    }
}
