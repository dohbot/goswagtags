package main

import "time"

type Name1 struct {
}

type Name2 struct {
} // test

// Name test3
type Name3 struct {
}

// Name test4-1
// Name test4-2
// Name test4-3
type Name4 struct {
} // @name Name4
// Name test4-5

// name5-1
type name5 struct {
} // name5-2

type GetAppListRes struct {
	RequestId string          `json:"requestId" xml:"requestId" form:"requestId"`
	Services  []GetServiceRes `json:"services" xml:"services" form:"services"`
} // aaa

type GetServiceRes struct {
	ResourceId  string     `json:"resourceId" xml:"resourceId" form:"resourceId"`
	User        string     `json:"user" xml:"user" form:"user"`
	Name        string     `json:"name" xml:"name" form:"name"`
	Queue       string     `json:"queue" xml:"queue" form:"queue"`
	Lifetime    int64      `json:"lifetime" xml:"lifetime" form:"lifetime"`
	Version     string     `json:"version" xml:"version" form:"version"`
	TypeName    string     `json:"typeName" xml:"typeName" form:"typeName"`
	TypeVersion string     `json:"typeVersion" xml:"typeVersion" form:"typeVersion"`
	CreatedAt   *time.Time `json:"createdAt" xml:"createdAt" form:"createdAt"`
	ModifiedAt  *time.Time `json:"modifiedAt" xml:"modifiedAt" form:"modifiedAt"`
	Enabled     bool       `json:"enabled" xml:"enabled" form:"enabled"`
} //
