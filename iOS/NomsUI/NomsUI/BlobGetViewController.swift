//
//  BlobGetViewController.swift
//  NomsUI
//
// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0
//

import UIKit
import Noms4ios

class BlobGetViewController: UIViewController {

    @IBOutlet weak var txtPath: UITextField!
    @IBOutlet weak var txtBlob: UITextView!
    @IBOutlet weak var imgBlob: UIImageView!

    override func viewDidLoad() {
        super.viewDidLoad()
        if txtPath.text == "" {
            txtPath.text = "\(demoNomsIoServer)::sf-bicycle-parking/raw.value"
        }
        txtBlob.text = ""
    }

    override func didReceiveMemoryWarning() {
        super.didReceiveMemoryWarning()
    }

    @IBAction func runBlobGet(_ sender: AnyObject) {
        var blobErr: NSError?
        var blobData: NSData?
        let ok = Noms4ios.GoNoms4iosBlobGet(txtPath.text, &blobData, &blobErr)
        if ok && blobData != nil {
            if let img = UIImage.init(data: blobData as! Data) {
                imgBlob.image = img
                imgBlob.isHidden = false
                txtBlob.isHidden = true
                txtBlob.text = nil
            } else {
                let dataString = String(data: blobData as! Data, encoding: .utf8)
                txtBlob.text = dataString
                txtBlob.isHidden = false
                imgBlob.isHidden = true
                imgBlob.image = nil
            }
        } else {
            txtBlob.text = "Error loading blob"
            txtBlob.isHidden = false
            imgBlob.isHidden = true
        }
    }
}
