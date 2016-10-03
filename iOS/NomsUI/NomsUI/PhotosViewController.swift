//
//  PhotosViewController.swift
//  NomsUI
//
// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0
//

import UIKit
import Noms4ios

class PhotosViewController: UICollectionViewController {

    var count = 0
    let photoDataset = "http://demo.noms.io/aa::photos"

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        count = 5
        //count = GoNoms4iosPhotoIndexGetCountOfDates(photoDataset)
        //count = GoNoms4iosGetPhotoCount(photoDataset, "byTag")
/*
        if let addr = developerNomsServerAddress {
            let dataset = "\(addr)::images"
            count = GoNoms4iosCountImagesInPhotos(dataset)
        }
 */
        collectionView?.reloadData()
    }

    override func viewDidLoad() {
        super.viewDidLoad()
    }

    override func numberOfSections(in collectionView: UICollectionView) -> Int {
        return count
    }

    override func collectionView(_ collectionView: UICollectionView, numberOfItemsInSection section: Int) -> Int {
//        let num = GoNoms4iosPhotoIndexGetCountAtDate(photoDataset, section)
//        return num
        return section*2
    }

    func fileByIndex(_ index: Int) -> URL {
        let filename = "\(index).jpg"
        return downloadDirectory().appendingPathComponent(filename)
    }

    func getImage(_ index: Int) -> UIImage? {
        let downloadPath = self.fileByIndex(index)
        // try to get from cache
        let fileExists = (try? downloadPath.checkResourceIsReachable()) ?? false
        if fileExists {
            var imageData: Data? = nil
            do {
                try imageData = Data.init(contentsOf: downloadPath)
            } catch {}
            if imageData != nil {
                if let img = UIImage.init(data: imageData!) {
                    return img
                }
            }
        } else {
            // download and cache
            var blobErr: NSError?
            var blobData: NSData?
            if let addr = developerNomsServerAddress {
                let path = "\(addr)::images.value[\(index)]"
                let ok = Noms4ios.GoNoms4iosBlobGet(path, &blobData, &blobErr)
                if ok {
                    do {
                        try blobData?.write(to: downloadPath)
                    } catch {}
                    if let img = UIImage.init(data: blobData as! Data) {
                        return img
                    }
                }
            }
        }
        return nil
    }

    override func collectionView(_ collectionView: UICollectionView,
                                 cellForItemAt indexPath: IndexPath) -> UICollectionViewCell {
        let cell = collectionView.dequeueReusableCell(withReuseIdentifier: "ImageCell",
                                                      for: indexPath) as! PhotosViewCell
        cell.backgroundColor = UIColor.black
        cell.imageView.image = nil

        /*
        DispatchQueue.global(qos: .background).async { () -> Void in
            if let img = self.getImage(indexPath.row) {
                DispatchQueue.main.async { () -> Void in
                    cell.imageView.image = img
                }
            }
        }
         */

        return cell
    }
}
