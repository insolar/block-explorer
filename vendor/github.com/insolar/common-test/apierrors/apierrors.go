// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md
//

package apierrors

type ApiError struct {
	Code        int
	Description string
}

var NoError = ApiError{}

var ExecutionError = ApiError{
	Code:        -31103,
	Description: "Execution error.",
}

var ParseError = ApiError{
	Code:        -31700,
	Description: "Parsing error on the server side: received an invalid JSON.",
}

var JsonValidationError = ApiError{
	Code:        -31600,
	Description: "The JSON received is not a valid request payload.",
}

var WrongGroupTypeError = ApiError{
	Code:        -31604,
	Description: "Wrong group type.",
}
