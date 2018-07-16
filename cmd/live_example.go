// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

/*
This is an cmd on how the Go library gomaasapi can be used to interact with
a real MAAS server.
Note that this is a provided only as an cmd and that real code should probably do something more sensible with errors than ignoring them or panicking.
*/
package main

import (
	"bytes"
	"fmt"
	"net/url"

	"encoding/json"

	"github.com/juju/gomaasapi/pkg/api/client"
	"github.com/juju/gomaasapi/pkg/api/v2"
)

var apiKey string
var apiURL string
var apiVersion string

func getParams() {
	fmt.Println("Warning: this will create a node on the MAAS server; it should be deleted at the end of the run but if something goes wrong, that test node might be left over.  You've been warned.")
	fmt.Print("Enter API key: ")
	_, err := fmt.Scanf("%s", &apiKey)
	if err != nil {
		panic(err)
	}
	fmt.Print("Enter API URL: ")
	_, err = fmt.Scanf("%s", &apiURL)
	if err != nil {
		panic(err)
	}

	fmt.Print("Enter API version: ")
	_, err = fmt.Scanf("%s", &apiVersion)
	if err != nil {
		panic(err)
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	getParams()

	// Create API server endpoint.
	authClient, err := client.NewAuthenticatedMAASClient(
		client.AddAPIVersionToURL(apiURL, apiVersion), apiKey)
	checkError(err)
	maas := client.NewMAAS(*authClient)

	// Exercise the API.
	ManipulateMachines(maas)
	ManipulateFiles(maas)

	fmt.Println("All done.")
}

// ManipulateFiles exercises the /api/1.0/files/ API endpoint.  Most precisely,
// it uploads a files and then fetches it, making sure the received content
// is the same as the one that was sent.
func ManipulateFiles(maas *client.MAASObject) {
	var err error
	maas.Client.Get()
	filesResource := maas.GetSubObject("files")

	fileContent := []byte("test file content")
	fileName := "filename"
	filesToUpload := map[string][]byte{"file": fileContent}

	// Upload a file.
	fmt.Println("Uploading a file...")
	_, err = filesResource.CallPostFiles("", url.Values{"filename": {fileName}}, filesToUpload)
	checkError(err)
	fmt.Println("FileInterface sent.")

	// Fetch the file.
	fmt.Println("Fetching the file...")
	fileResult, err := filesResource.CallGet("get", url.Values{"filename": {fileName}})
	checkError(err)

	receivedFileContent := fileResult.Values

	if bytes.Compare(receivedFileContent, fileContent) != 0 {
		panic("Received content differs from the content sent!")
	}
	fmt.Println("Got file.")

	// Fetch list of filesResource.
	var listFiles []maasapiv2.File
	listFilesObj, err := filesResource.CallGet("", url.Values{})
	checkError(err)
	json.Unmarshal(listFilesObj.Values, &listFiles)

	fmt.Printf("We've got %v file(s)\n", len(listFiles))

	// Delete the file.
	fmt.Println("Deleting the file...")

	fileObject := listFiles[0]
	err = fileObject.Delete()
	checkError(err)

	// Count the filesResource.
	listFilesObj, err = filesResource.CallGet("", url.Values{})
	checkError(err)
	json.Unmarshal(listFilesObj.Values, &listFiles)
	fmt.Printf("We've got %v file(s)\n", len(listFiles))
}

// ManipulateFiles exercises the /api/1.0/nodes/ API endpoint.  Most precisely,
// it lists the existing nodes, creates a new node, updates it and then
// deletes it.
func ManipulateMachines(maas *client.MAASObject) {
	var machines []maasapiv2.Machine
	var machines2 []maasapiv2.Machine
	var machines3 []maasapiv2.Machine
	var newMachine maasapiv2.Machine
	var newMachine2 maasapiv2.Machine

	machinesRequestObj := maas.GetSubObject("machines")

	// List nodes.
	fmt.Println("Fetching list of machines...")
	machinesObject, err := machinesRequestObj.CallGet("", url.Values{})
	checkError(err)

	err = json.Unmarshal(machinesObject.Values, &machines)
	checkError(err)

	fmt.Printf("Got list of %v nodes\n", len(machines))
	for index, node := range machines {
		fmt.Printf("Machine #%d is named '%v' (%v)\n", index, node.Hostname, node.ResourceURI)
	}

	// Create a node.
	fmt.Println("Creating a new machine...")
	params := url.Values{"architecture": {"amd64/generic"}, "mac_addresses": {"AA:BB:CC:DD:EE:FF"}, "power_type": {"manual"}}
	newMachineObj, err := machinesRequestObj.CallPost("", params)
	checkError(err)

	err = json.Unmarshal(newMachineObj.Values, &newMachine)
	checkError(err)
	fmt.Printf("New node created: %s (%s)\n", newMachine.Hostname, newMachine.ResourceURI)

	// Update the new node.
	fmt.Println("Updating the new node...")
	updateParams := url.Values{"hostname": {"mynewname"}}
	newNodeObj2, err := newMachineObj.Update(updateParams)
	checkError(err)

	err = json.Unmarshal(newNodeObj2.Values, &newMachine2)
	checkError(err)
	fmt.Printf("New machine updated, now named: %s\n", newMachine2.Hostname)

	// Count the nodes.
	listNodeObjects2, err := machinesRequestObj.CallGet("", url.Values{})
	checkError(err)

	err = json.Unmarshal(listNodeObjects2.Values, &machines2)
	checkError(err)

	checkError(err)
	fmt.Printf("We've got %v nodes\n", len(machines2))

	// Delete the new node.
	fmt.Println("Deleting the new node...")
	errDelete := newMachineObj.Delete()
	checkError(errDelete)

	// Count the nodes.
	listNodeObjects3, err := machinesRequestObj.CallGet("", url.Values{})
	checkError(err)

	err = json.Unmarshal(listNodeObjects3.Values, &machines3)
	checkError(err)
	checkError(err)
	fmt.Printf("We've got %v nodes\n", len(machines3))
}
