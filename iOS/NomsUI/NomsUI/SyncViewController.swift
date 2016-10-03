//
//  SyncViewController.swift
//  NomsUI
//
// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0
//

import UIKit
import Noms4ios

class SyncViewController: UIViewController {

    @IBOutlet weak var txtSource: UITextField!
    @IBOutlet weak var txtDestination: UITextField!

    override func viewDidLoad() {
        super.viewDidLoad()
        if txtSource.text == "" {
            txtSource.text = "\(demoNomsIoServer)::sf-bicycle-parking"
        }
        if txtDestination.text == "" {
            txtDestination.text = "sfbike"
        }
    }

    override func didReceiveMemoryWarning() {
        super.didReceiveMemoryWarning()
    }

    @IBAction func runSync(_ sender: AnyObject) {
        var err: NSError?
        let tmpLocation = stripFileUrlScheme(localDatasetsDirectory())
        let dst = "ldb:\(tmpLocation)::\(txtDestination.text!)"
        let ok = Noms4ios.GoNoms4iosSync(txtSource.text!, dst, 512, &err)
        print("sync returned \(ok) and \(err)")
    }
}
