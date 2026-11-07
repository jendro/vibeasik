package api

import "github.com/vibeasik/vibeasik/pkg/payload"

// Alias tipis ke pkg/payload (implementasi tunggal, dipakai juga oleh vibeasik-pro dan cloud).
const (
	maxName  = payload.MaxName
	maxLabel = payload.MaxLabel
	maxText  = payload.MaxText
	maxPath  = payload.MaxPath
	maxURL   = payload.MaxURL
)

var (
	badPayload       = payload.Bad
	bindJSON         = payload.BindJSON
	bindJSONRequired = payload.BindJSONRequired
	firstProblem     = payload.FirstProblem
	needText         = payload.NeedText
	maxLen           = payload.MaxLen
	oneOf            = payload.OneOf
	needOneOf        = payload.NeedOneOf
	needID           = payload.NeedID
	absPath          = payload.AbsPath
	paramID          = payload.ParamID
)
