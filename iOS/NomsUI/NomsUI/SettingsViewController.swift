//
//  SettingsViewController.swift
//  NomsUI
//
// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0
//

import UIKit
import Noms4ios

class SettingsViewController: UITableViewController {

    @IBOutlet weak var lblVersion: UILabel!
    @IBOutlet weak var txtServerAddress: UITextField!

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        if let addr = developerNomsServerAddress {
            txtServerAddress.text = addr
        }
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        var str = Noms4ios.GoNoms4iosVersion()!
        str = str.replacingOccurrences(of: "\n", with: ", ")
        lblVersion.text = str.substring(to: str.index(str.endIndex, offsetBy: -2))
    }

    override func didReceiveMemoryWarning() {
        super.didReceiveMemoryWarning()
    }

    @IBAction func doneEditing(_ sender: AnyObject) {
        developerNomsServerAddress = txtServerAddress.text
    }
}
