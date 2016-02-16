// Copyright 2015 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

func getSpacesEndpoint(version string) string {
	return fmt.Sprintf("/api/%s/spaces/", version)
}

// Space is the MAAS API space representation
type Space struct {
	Name        string   `json:"name"`
	Subnets     []Subnet `json:"subnets"`
	ResourceURI string   `json:"resource_uri"`
	ID          uint     `json:"id"`
}

// spacesHandler handles requests for '/api/<version>/spaces/'.
func spacesHandler(server *TestServer, w http.ResponseWriter, r *http.Request) {
	var err error
	spacesURLRE := regexp.MustCompile(`/spaces/(.+?)/`)
	spacesURLMatch := spacesURLRE.FindStringSubmatch(r.URL.Path)
	spacesURL := getSpacesEndpoint(server.version)

	var ID uint
	var gotID bool
	if spacesURLMatch != nil {
		ID, err = NameOrIDToID(spacesURLMatch[1], server.spaceNameToID, 1, uint(len(server.spaces)))

		if err != nil {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		gotID = true
	}

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if len(server.spaces) == 0 {
			// Until a space is registered, behave as if the endpoint
			// does not exist. This way we can simulate older MAAS
			// servers that do not support spaces.
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		if r.URL.Path == spacesURL {
			var spaces []*Space
			// Iterating by id rather than a dictionary iteration
			// preserves the order of the spaces in the result.
			for i := uint(1); i < server.nextSpace; i++ {
				s, ok := server.spaces[i]
				if ok {
					server.setSubnetsOnSpace(s)
					spaces = append(spaces, s)
				}
			}
			err = json.NewEncoder(w).Encode(spaces)
		} else if gotID == false {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			err = json.NewEncoder(w).Encode(server.spaces[ID])
		}
		checkError(err)
	case "POST":
		//server.NewSpace(r.Body)
	case "PUT":
		//server.UpdateSpace(r.Body)
	case "DELETE":
		delete(server.spaces, ID)
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

// CreateSpace is used to create new spaces on the server.
type CreateSpace struct {
	Name string `json:"name"`
}

func decodePostedSpace(spaceJSON io.Reader) CreateSpace {
	var postedSpace CreateSpace
	decoder := json.NewDecoder(spaceJSON)
	err := decoder.Decode(&postedSpace)
	checkError(err)
	return postedSpace
}

// NewSpace creates a space in the test server
func (server *TestServer) NewSpace(spaceJSON io.Reader) *Space {
	postedSpace := decodePostedSpace(spaceJSON)
	newSpace := &Space{Name: postedSpace.Name}
	newSpace.ID = server.nextSpace
	newSpace.ResourceURI = fmt.Sprintf("/api/%s/spaces/%d/", server.version, int(server.nextSpace))
	server.spaces[server.nextSpace] = newSpace
	server.spaceNameToID[newSpace.Name] = newSpace.ID

	server.nextSpace++
	return newSpace
}

// setSubnetsOnSpace fetches the subnets for the specified space and adds them
// to it.
func (server *TestServer) setSubnetsOnSpace(space *Space) {
	subnets := []Subnet{}
	for i := uint(1); i < server.nextSubnet; i++ {
		subnet, ok := server.subnets[i]
		if ok && subnet.Space == space.Name {
			subnets = append(subnets, subnet)
		}
	}
	space.Subnets = subnets
}
