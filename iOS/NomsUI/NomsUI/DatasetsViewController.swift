//
//  DatasetsViewController.swift
//  NomsUI
//
// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0
//

import UIKit
import Noms4ios

class DatasetsViewController: UITableViewController {

    var datasets: [String: [String]] = [:]
    let localDataServer = "_local"

    func loadDatasets() {
        let localDatasets = datasetsAvailable(stripFileUrlScheme(localDatasetsDirectory()))
        if localDatasets.count > 0 {
            datasets[localDataServer] = localDatasets
        } else {
            datasets.removeValue(forKey: localDataServer)
        }
        if let addr = developerNomsServerAddress {
            let developerDatasets = datasetsAvailable(addr)
            if developerDatasets.count > 0 {
                datasets[addr] = developerDatasets
            } else {
                datasets.removeValue(forKey: addr)
            }
        }
        datasets[demoNomsIoServer] = datasetsAvailable(demoNomsIoServer)
    }

    func serverByIndex(_ index: Int) -> String {
        let servers = datasets.keys.sorted()
        return servers[index]
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        loadDatasets()
        tableView.reloadData()
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        //loadDatasets()
        self.refreshControl?.addTarget(self, action: #selector(self.handleRefresh(_:)), for: UIControlEvents.valueChanged)
    }

    override func didReceiveMemoryWarning() {
        super.didReceiveMemoryWarning()
    }

    func handleRefresh(_ refreshControl: UIRefreshControl) {
        loadDatasets()
        self.tableView.reloadData()
        refreshControl.endRefreshing()
    }

    override func numberOfSections(in tableView: UITableView) -> Int {
        return datasets.count
    }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        let servers = datasets.keys.sorted()
        return (datasets[servers[section]]?.count)!
    }

    override func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        return serverByIndex(section)
    }

    override func tableView(_ tableView: UITableView, canEditRowAt indexPath: IndexPath) -> Bool {
        let server = serverByIndex(indexPath.section)
        return server != demoNomsIoServer
    }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "DataSetCell", for: indexPath)

        let server = serverByIndex(indexPath.section)
        let datasetName = (datasets[server]?[indexPath.row])!

        cell.textLabel?.text = datasetName
        return cell
    }

    override func tableView(_ tableView: UITableView, commit editingStyle: UITableViewCellEditingStyle, forRowAt indexPath: IndexPath) {
        if editingStyle == .delete {
            var server = serverByIndex(indexPath.section)
            let datasetName = (datasets[server]?[indexPath.row])!
            if server == localDataServer {
                server = stripFileUrlScheme(localDatasetsDirectory())
            }
            GoNoms4iosDsDelete("\(server)::\(datasetName)")

            tableView.reloadData()
            //tableView.deleteRows(at: [indexPath], with: .automatic)
        }
    }
}
