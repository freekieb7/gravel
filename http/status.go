// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

const (
	StatusContinue           uint16 = 100 // RFC 7231, 6.2.1
	StatusSwitchingProtocols uint16 = 101 // RFC 7231, 6.2.2
	StatusProcessing         uint16 = 102 // RFC 2518, 10.1
	StatusEarlyHints         uint16 = 103 // RFC 8297

	StatusOK                   uint16 = 200 // RFC 7231, 6.3.1
	StatusCreated              uint16 = 201 // RFC 7231, 6.3.2
	StatusAccepted             uint16 = 202 // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo uint16 = 203 // RFC 7231, 6.3.4
	StatusNoContent            uint16 = 204 // RFC 7231, 6.3.5
	StatusResetContent         uint16 = 205 // RFC 7231, 6.3.6
	StatusPartialContent       uint16 = 206 // RFC 7233, 4.1
	StatusMultiStatus          uint16 = 207 // RFC 4918, 11.1
	StatusAlreadyReported      uint16 = 208 // RFC 5842, 7.1
	StatusIMUsed               uint16 = 226 // RFC 3229, 10.4.1

	StatusMultipleChoices   uint16 = 300 // RFC 7231, 6.4.1
	StatusMovedPermanently  uint16 = 301 // RFC 7231, 6.4.2
	StatusFound             uint16 = 302 // RFC 7231, 6.4.3
	StatusSeeOther          uint16 = 303 // RFC 7231, 6.4.4
	StatusNotModified       uint16 = 304 // RFC 7232, 4.1
	StatusUseProxy          uint16 = 305 // RFC 7231, 6.4.5
	_                       uint16 = 306 // RFC 7231, 6.4.6 (Unused)
	StatusTemporaryRedirect uint16 = 307 // RFC 7231, 6.4.7
	StatusPermanentRedirect uint16 = 308 // RFC 7538, 3

	StatusBadRequest                   uint16 = 400 // RFC 7231, 6.5.1
	StatusUnauthorized                 uint16 = 401 // RFC 7235, 3.1
	StatusPaymentRequired              uint16 = 402 // RFC 7231, 6.5.2
	StatusForbidden                    uint16 = 403 // RFC 7231, 6.5.3
	StatusNotFound                     uint16 = 404 // RFC 7231, 6.5.4
	StatusMethodNotAllowed             uint16 = 405 // RFC 7231, 6.5.5
	StatusNotAcceptable                uint16 = 406 // RFC 7231, 6.5.6
	StatusProxyAuthRequired            uint16 = 407 // RFC 7235, 3.2
	StatusRequestTimeout               uint16 = 408 // RFC 7231, 6.5.7
	StatusConflict                     uint16 = 409 // RFC 7231, 6.5.8
	StatusGone                         uint16 = 410 // RFC 7231, 6.5.9
	StatusLengthRequired               uint16 = 411 // RFC 7231, 6.5.10
	StatusPreconditionFailed           uint16 = 412 // RFC 7232, 4.2
	StatusRequestEntityTooLarge        uint16 = 413 // RFC 7231, 6.5.11
	StatusRequestURITooLong            uint16 = 414 // RFC 7231, 6.5.12
	StatusUnsupportedMediaType         uint16 = 415 // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable uint16 = 416 // RFC 7233, 4.4
	StatusExpectationFailed            uint16 = 417 // RFC 7231, 6.5.14
	StatusTeapot                       uint16 = 418 // RFC 7168, 2.3.3
	StatusMisdirectedRequest           uint16 = 421 // RFC 7540, 9.1.2
	StatusUnprocessableEntity          uint16 = 422 // RFC 4918, 11.2
	StatusLocked                       uint16 = 423 // RFC 4918, 11.3
	StatusFailedDependency             uint16 = 424 // RFC 4918, 11.4
	StatusUpgradeRequired              uint16 = 426 // RFC 7231, 6.5.15
	StatusPreconditionRequired         uint16 = 428 // RFC 6585, 3
	StatusTooManyRequests              uint16 = 429 // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge  uint16 = 431 // RFC 6585, 5
	StatusUnavailableForLegalReasons   uint16 = 451 // RFC 7725, 3

	StatusInternalServerError           uint16 = 500 // RFC 7231, 6.6.1
	StatusNotImplemented                uint16 = 501 // RFC 7231, 6.6.2
	StatusBadGateway                    uint16 = 502 // RFC 7231, 6.6.3
	StatusServiceUnavailable            uint16 = 503 // RFC 7231, 6.6.4
	StatusGatewayTimeout                uint16 = 504 // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       uint16 = 505 // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         uint16 = 506 // RFC 2295, 8.1
	StatusInsufficientStorage           uint16 = 507 // RFC 4918, 11.5
	StatusLoopDetected                  uint16 = 508 // RFC 5842, 7.2
	StatusNotExtended                   uint16 = 510 // RFC 2774, 7
	StatusNetworkAuthenticationRequired uint16 = 511 // RFC 6585, 6
)

var (
	unknownStatusCode = "Unknown Status Code"

	statusMessages = []string{
		StatusContinue:           "Continue",
		StatusSwitchingProtocols: "Switching Protocols",
		StatusProcessing:         "Processing",
		StatusEarlyHints:         "Early Hints",

		StatusOK:                   "OK",
		StatusCreated:              "Created",
		StatusAccepted:             "Accepted",
		StatusNonAuthoritativeInfo: "Non-Authoritative Information",
		StatusNoContent:            "No Content",
		StatusResetContent:         "Reset Content",
		StatusPartialContent:       "Partial Content",
		StatusMultiStatus:          "Multi-Status",
		StatusAlreadyReported:      "Already Reported",
		StatusIMUsed:               "IM Used",

		StatusMultipleChoices:   "Multiple Choices",
		StatusMovedPermanently:  "Moved Permanently",
		StatusFound:             "Found",
		StatusSeeOther:          "See Other",
		StatusNotModified:       "Not Modified",
		StatusUseProxy:          "Use Proxy",
		StatusTemporaryRedirect: "Temporary Redirect",
		StatusPermanentRedirect: "Permanent Redirect",

		StatusBadRequest:                   "Bad Request",
		StatusUnauthorized:                 "Unauthorized",
		StatusPaymentRequired:              "Payment Required",
		StatusForbidden:                    "Forbidden",
		StatusNotFound:                     "Not Found",
		StatusMethodNotAllowed:             "Method Not Allowed",
		StatusNotAcceptable:                "Not Acceptable",
		StatusProxyAuthRequired:            "Proxy Authentication Required",
		StatusRequestTimeout:               "Request Timeout",
		StatusConflict:                     "Conflict",
		StatusGone:                         "Gone",
		StatusLengthRequired:               "Length Required",
		StatusPreconditionFailed:           "Precondition Failed",
		StatusRequestEntityTooLarge:        "Request Entity Too Large",
		StatusRequestURITooLong:            "Request URI Too Long",
		StatusUnsupportedMediaType:         "Unsupported Media Type",
		StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
		StatusExpectationFailed:            "Expectation Failed",
		StatusTeapot:                       "I'm a teapot",
		StatusMisdirectedRequest:           "Misdirected Request",
		StatusUnprocessableEntity:          "Unprocessable Entity",
		StatusLocked:                       "Locked",
		StatusFailedDependency:             "Failed Dependency",
		StatusUpgradeRequired:              "Upgrade Required",
		StatusPreconditionRequired:         "Precondition Required",
		StatusTooManyRequests:              "Too Many Requests",
		StatusRequestHeaderFieldsTooLarge:  "Request Header Fields Too Large",
		StatusUnavailableForLegalReasons:   "Unavailable For Legal Reasons",

		StatusInternalServerError:           "Internal Server Error",
		StatusNotImplemented:                "Not Implemented",
		StatusBadGateway:                    "Bad Gateway",
		StatusServiceUnavailable:            "Service Unavailable",
		StatusGatewayTimeout:                "Gateway Timeout",
		StatusHTTPVersionNotSupported:       "HTTP Version Not Supported",
		StatusVariantAlsoNegotiates:         "Variant Also Negotiates",
		StatusInsufficientStorage:           "Insufficient Storage",
		StatusLoopDetected:                  "Loop Detected",
		StatusNotExtended:                   "Not Extended",
		StatusNetworkAuthenticationRequired: "Network Authentication Required",
	}
)
