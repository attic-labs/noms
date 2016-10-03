//
//  BlobPutViewController.swift
//  NomsUI
//
// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0
//

import UIKit
import Noms4ios

class BlobPutViewController: UIViewController, UIImagePickerControllerDelegate, UINavigationControllerDelegate {

    @IBOutlet weak var imageView: UIImageView!
    @IBOutlet weak var btnSaveToAttic: UIButton!

    let imagePicker = UIImagePickerController()
    var imageResourceUrl: URL!

    override func viewDidLoad() {
        super.viewDidLoad()
        imagePicker.delegate = self
        btnSaveToAttic.isHidden = true
    }

    override func didReceiveMemoryWarning() {
        super.didReceiveMemoryWarning()
    }

    func imagePickerController(_ picker: UIImagePickerController, didFinishPickingMediaWithInfo info: [String : Any]) {
        imageResourceUrl = info[UIImagePickerControllerReferenceURL] as? URL
        imageView.image = info[UIImagePickerControllerOriginalImage] as? UIImage
        if developerNomsServerAddress != nil {
            btnSaveToAttic.isHidden = false
        }
        self.dismiss(animated: true, completion: nil);
    }

    @IBAction func saveToAttic(_ sender: AnyObject) {
        let jpegData = UIImageJPEGRepresentation(imageView.image!, 1.0)!
        let path = uploadDirectory().appendingPathComponent("image.jpg")
        do {
            try jpegData.write(to: path)
        } catch let error as NSError {
            print("Could not write file", error.localizedDescription)
        }
        let addr = developerNomsServerAddress!
        let dataset = "\(addr)::images"
        GoNoms4iosAddImageToPhotos(stripFileUrlScheme(path), imageResourceUrl.absoluteString, dataset)
        deleteFile(path)
        btnSaveToAttic.isHidden = true
        imageResourceUrl = nil
        imageView.image = nil
    }

    @IBAction func openCamera(_ sender: AnyObject) {
        btnSaveToAttic.isHidden = true
        if UIImagePickerController.isSourceTypeAvailable(.camera) {
            imagePicker.sourceType = .camera
            imagePicker.cameraCaptureMode = .photo
            self.present(imagePicker, animated: true, completion: nil)
        }
    }

    @IBAction func openPhotoLibrary(_ sender: AnyObject) {
        btnSaveToAttic.isHidden = true
        if UIImagePickerController.isSourceTypeAvailable(.photoLibrary) {
            imagePicker.sourceType = .photoLibrary
            self.present(imagePicker, animated: true, completion: nil)
        }
    }
}
