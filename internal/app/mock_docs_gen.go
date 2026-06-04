package app

import (
	"encoding/json"
	"sync"

	echodoc0 "openapi-mock/internal/generated/echo"
	petstoredoc1 "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/mgmt"
)

func MockDocs() []mgmt.MockDoc {
	return []mgmt.MockDoc{
		func() mgmt.MockDoc {
			var once sync.Once
			var cached []byte
			var cachedErr error
			return mgmt.MockDoc{
				APIName:    "echo",
				APIVersion: "",
				Title:      "Echo API",
				SpecJSON: func() ([]byte, error) {
					once.Do(func() {
						doc, err := echodoc0.GetSwagger()
						if err != nil {
							cachedErr = err
							return
						}
						cached, cachedErr = json.Marshal(doc)
					})
					return cached, cachedErr
				},
			}
		}(),
		func() mgmt.MockDoc {
			var once sync.Once
			var cached []byte
			var cachedErr error
			return mgmt.MockDoc{
				APIName:    "petstore",
				APIVersion: "",
				Title:      "Petstore API",
				SpecJSON: func() ([]byte, error) {
					once.Do(func() {
						doc, err := petstoredoc1.GetSwagger()
						if err != nil {
							cachedErr = err
							return
						}
						cached, cachedErr = json.Marshal(doc)
					})
					return cached, cachedErr
				},
			}
		}(),
	}
}
