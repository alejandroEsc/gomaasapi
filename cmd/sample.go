// Copyright 2012-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

/*
This is an cmd on how the Go library gomaasapi can be used to interact with
a real maas server.
Note that this is a provided only as an cmd and that real code should probably do something more sensible with errors than ignoring them or panicking.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/juju/gomaasapi/pkg/api/maas"
	"github.com/juju/gomaasapi/pkg/api/v2"
	"github.com/spf13/viper"
)

const (
	apiKeyKey         = "api_key"
	maasURLKey        = "api_url"
	maasAPIVersionKey = "api_version"
)

var apiKey string
var apiURL string
var apiVersion string

// Init initializes the environment variables to be used by the app
func Init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("maas")
	viper.BindEnv(maasURLKey)
	viper.BindEnv(apiKeyKey)
	viper.BindEnv(maasAPIVersionKey)
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func main() {
	Init()
	apiKey := viper.GetString(apiKeyKey)
	apiURL := viper.GetString(maasURLKey)
	apiVersion := viper.GetString(maasAPIVersionKey)

	fmt.Printf("values are: %s %s %s\n", apiKey, apiURL, apiVersion)

	m, err := maas.NewMASS(apiURL, apiVersion, apiKey)
	checkError(err)

	files(m)
	machines(m)
	nodes(m)

	fmt.Println("All done.")
}

func machines(maas *maas.MAAS) {

	params := maasapiv2.MachinesParams(maasapiv2.MachinesArgs{})
	rawMachines, err := maas.Get("machines", "", params.Values)
	checkError(err)

	var machines []maasapiv2.Machine
	err = json.Unmarshal(rawMachines, &machines)
	checkError(err)

	fmt.Printf("\nGot list of %v machines\n", len(machines))
	for index, machine := range machines {
		fmt.Printf("Machine #%d is named '%v' (%v)\n", index, machine.Hostname, machine.ResourceURI)
	}

}

func nodes(maas *maas.MAAS) {
	rawNodes, err := maas.Get("nodes", "", nil)
	checkError(err)

	fmt.Printf("\n%s\n", string(rawNodes))
}

// ManipulateFiles exercises the /api/1.0/files/ API endpoint.  Most precisely,
// it uploads a files and then fetches it, making sure the received content
// is the same as the one that was sent.
func files(maas *maas.MAAS) {
	var err error

	fileContent := []byte("test file content")
	fileName := "myfilename"

	// Upload a file.
	fmt.Println("Uploading a file...")
	params := maasapiv2.FileParams(maasapiv2.AddFileArgs{Filename: fileName})

	_, err = maas.PostFile("files", "", params.Values, fileContent)
	checkError(err)
	fmt.Println("File sent.")

	// Fetch the file.
	fileBytes, err := maas.Get("files/"+fileName, "", nil)
	checkError(err)
	var file maasapiv2.File
	err = json.Unmarshal(fileBytes, &file)

	if bytes.Compare([]byte(file.Content), fileContent) != 0 {
		maas.Delete(file.ResourceURI)
		panic("Received content differs from the content sent!")
	}
	fmt.Println("Got file.")

	// Fetch list of filesResource.
	var listFiles []maasapiv2.File
	listBytes, err := maas.Get("files", "", nil)
	err = json.Unmarshal(listBytes, &listFiles)
	checkError(err)

	fmt.Printf("We've got %v file(s)\n", len(listFiles))

	// Delete the file.
	fmt.Println("Deleting the file...")

	f := listFiles[0]
	err = maas.Delete(f.ResourceURI)
	checkError(err)

	// Count the filesResource.
	listBytes, err = maas.Get("files", "", nil)
	checkError(err)
	json.Unmarshal(listBytes, &listFiles)
	fmt.Printf("We've got %v file(s)\n", len(listFiles))
}
