//
//  FacebookPhotos.swift
//  NomsUI
//
//  Created by Mike Gray on 9/30/16.
//  Copyright © 2016 Mike Gray. All rights reserved.
//

import Foundation
import Accounts
import Social

class FacebookPhotos {
/*
    func doSomething() {
        var accountStore = ACAccountStore()
        var accountType = accountStore.accountType(withAccountTypeIdentifier: ACAccountTypeIdentifierFacebook)

        var postingOptions = [ACFacebookAppIdKey: "<YOUR FACEBOOK APP ID KEY HERE>",
                              ACFacebookPermissionsKey: ["email"],
                              ACFacebookAudienceKey: ACFacebookAudienceFriends] as [String : Any]

        accountStore.requestAccessToAccounts(with: accountType, options: postingOptions) { success, error in
            if success {

                var options = [ACFacebookAppIdKey: "<YOUR FACEBOOK APP ID KEY HERE>",
                               ACFacebookPermissionsKey: ["publish_actions"],
                               ACFacebookAudienceKey: ACFacebookAudienceFriends] as [String : Any]
                accountStore.requestAccessToAccounts(with: accountType, options: options) { success, error in
                    if success {
                        var accountsArray = accountStore.accounts(with: accountType)

                        if (accountsArray?.count)! > 0 {
                            var facebookAccount = accountsArray?[0] as! ACAccount

                            var parameters = Dictionary<String, AnyObject>()
                            parameters["access_token"] = facebookAccount.credential.oauthToken as AnyObject?
                            parameters["message"] = "My first Facebook post from iOS 8" as AnyObject?

                            var feedURL = NSURL(string: "https://graph.facebook.com/me/feed")

                            let postRequest = SLRequest(forServiceType: SLServiceTypeFacebook,
                                                        requestMethod: SLRequestMethod.POST,
                                                        url: feedURL as URL!,
                                                        parameters: parameters)
                            postRequest?.performRequestWithHandler(
                                {(responseData: NSData!, urlResponse: HTTPURLResponse!, error: NSError!) -> Void in
                                    print("Facebook HTTP response: \(urlResponse.statusCode)")
                                })
                        }
                    } else {
                        print("Access denied - \(error?.localizedDescription)")
                    }
                }
            } else {
                print("Access denied - \(error?.localizedDescription)")
            }
        }
    }
*/

    
/*
    let account = ACAccountStore()
let accountType = account.accountTypeWithAccountTypeIdentifier(
    ACAccountTypeIdentifierTwitter)

account.requestAccessToAccountsWithType(accountType, options: nil,
                                        completion: {(success: Bool, error: NSError!) -> Void in

                                            if success {
                                                let arrayOfAccounts =
                                                    account.accountsWithAccountType(accountType)

                                                if arrayOfAccounts.count > 0 {
                                                    let twitterAccount = arrayOfAccounts.last as! ACAccount
                                                    let message = ["status" : "My first post from iOS 8"]
                                                    let requestURL = NSURL(string:
                                                        "https://api.twitter.com/1.1/statuses/update.json")
                                                    let postRequest = SLRequest(forServiceType:
                                                        SLServiceTypeTwitter,
                                                                                requestMethod: SLRequestMethod.POST, 
                                                                                URL: requestURL, 
                                                                                parameters: message)
                                                    
                                                    postRequest.account = twitterAccount
                                                    
                                                    postRequest.performRequestWithHandler({
                                                        (responseData: NSData!, 
                                                        urlResponse: NSHTTPURLResponse!, 
                                                        error: NSError!) -> Void in
                                                        
                                                        if let err = error {
                                                            println("Error : \(err.localizedDescription)")
                                                        }
                                                        println("Twitter HTTP response \(urlResponse.statusCode)")
                                                    })
                                                }
                                            }
})
*/

}
