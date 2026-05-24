package mgmt

import "sort"

// MockDoc describes a mock API OpenAPI document exposed by the management API.
type MockDoc struct {
	APIName    string
	APIVersion string
	Title      string
	SpecJSON   func() ([]byte, error)
}

type mockDocsIndex struct {
	all     []MockDoc
	byName  map[string][]MockDoc
	byExact map[string]MockDoc
}

func newMockDocsIndex(docs []MockDoc) *mockDocsIndex {
	out := &mockDocsIndex{
		all:     make([]MockDoc, 0, len(docs)),
		byName:  make(map[string][]MockDoc),
		byExact: make(map[string]MockDoc),
	}
	for _, d := range docs {
		if d.APIName == "" || d.SpecJSON == nil {
			continue
		}
		out.all = append(out.all, d)
		out.byName[d.APIName] = append(out.byName[d.APIName], d)
		out.byExact[d.APIName+"\x00"+d.APIVersion] = d
	}
	for name := range out.byName {
		sort.Slice(out.byName[name], func(i, j int) bool {
			return out.byName[name][i].APIVersion < out.byName[name][j].APIVersion
		})
	}
	sort.Slice(out.all, func(i, j int) bool {
		if out.all[i].APIName == out.all[j].APIName {
			return out.all[i].APIVersion < out.all[j].APIVersion
		}
		return out.all[i].APIName < out.all[j].APIName
	})
	return out
}

func (i *mockDocsIndex) list() []MockDoc {
	out := make([]MockDoc, len(i.all))
	copy(out, i.all)
	return out
}

func (i *mockDocsIndex) find(apiName, apiVersion string) (MockDoc, bool) {
	doc, ok := i.byExact[apiName+"\x00"+apiVersion]
	return doc, ok
}

func (i *mockDocsIndex) resolve(apiName string) (doc MockDoc, hasDoc bool, versions []MockDoc, ambiguous bool) {
	items := i.byName[apiName]
	if len(items) == 0 {
		return MockDoc{}, false, nil, false
	}
	if unversioned, ok := i.find(apiName, ""); ok {
		return unversioned, true, nil, false
	}
	if len(items) == 1 {
		return items[0], true, nil, false
	}
	out := make([]MockDoc, len(items))
	copy(out, items)
	return MockDoc{}, false, out, true
}
