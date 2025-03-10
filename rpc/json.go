// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/sense"
)

const (
	JsonrpcVersion           = "2.0"
	ServiceMethodSeparator   = "_"
	SubscribeMethodSuffix    = "_subscribe"
	UnsubscribeMethodSuffix  = "_unsubscribe"
	NotificationMethodSuffix = "_subscription"
)

type JsonRequest = jsonRequest
type JsonSuccessResponse = jsonSuccessResponse
type JsonErrResponse = jsonErrResponse
type JsonSubscription = jsonSubscription
type JsonNotification = jsonNotification

type jsonRequest struct {
	Method  string          `json:"method"`
	Version string          `json:"jsonrpc"`
	Id      json.RawMessage `json:"id,omitempty"`
	Payload json.RawMessage `json:"params,omitempty"`
}

type jsonSuccessResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
type JsonError = jsonError

type jsonErrResponse struct {
	Version string      `json:"jsonrpc"`
	Id      interface{} `json:"id,omitempty"`
	Error   jsonError   `json:"error"`
}

type jsonSubscription struct {
	Subscription string      `json:"subscription"`
	Result       interface{} `json:"result,omitempty"`
}

type jsonNotification struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  jsonSubscription `json:"params"`
}

type JsonCodec = jsonCodec

// jsonCodec reads and writes JSON-RPC messages to the underlying connection. It
// also has support for parsing arguments and serializing (result) objects.
type jsonCodec struct {
	closer sync.Once                 // close closed channel once
	closed chan interface{}          // closed on Close
	decMu  sync.Mutex                // guards d
	decode func(v interface{}) error // decodes incoming requests
	encMu  sync.Mutex                // guards e
	encode func(v interface{}) error // encodes responses
	rw     CodecConn                 // connection
}

type CodecConn interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}

func (err *jsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}
	return err.Message
}

func (err *jsonError) ErrorCode() int {
	return err.Code
}

var _ ServerCodec = (*JsonCodec)(nil)

// NewCodec creates a new RPC server codec with support for JSON-RPC 2.0 based
// on explicitly given encoding and decoding methods.
func NewCodec(rwc CodecConn, encode, decode func(v interface{}) error) *JsonCodec {
	return &jsonCodec{
		closed: make(chan interface{}),
		encode: encode,
		decode: decode,
		rw:     rwc,
	}
}

var _DEBUG_RPC_REQUESTS = sense.Getenv("DEBUG_RPC_REQUESTS") == "1"
var _DEBUG_RPC_RESPONSES = sense.Getenv("DEBUG_RPC_RESPONSES") == "1"
var _DEBUG_RPC = sense.Getenv("DEBUG_RPC") == "1"

// NewJSONCodec creates a new RPC server codec with support for JSON-RPC 2.0
func NewJSONCodec(rwc CodecConn) *JsonCodec {
	var w io.Writer = rwc
	if _DEBUG_RPC_RESPONSES {
		w = io.MultiWriter(rwc, os.Stdout) // also log responses to stdout
	}
	var r io.Reader = rwc
	if _DEBUG_RPC_REQUESTS {
		r = io.TeeReader(rwc, os.Stdout) // also log requests to stdout
	}
	enc := json.NewEncoder(w)
	dec := json.NewDecoder(r)
	dec.UseNumber()

	return &JsonCodec{
		closed: make(chan interface{}),
		encode: enc.Encode,
		decode: dec.Decode,
		rw:     rwc,
	}
}

// isBatch returns true when the first non-whitespace characters is '['
func isBatch(msg json.RawMessage) bool {
	for _, c := range msg {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// ReadRequestHeaders will read new requests without parsing the arguments. It will
// return a collection of requests, an indication if these requests are in batch
// form or an error when the incoming message could not be read/parsed.
func (c *jsonCodec) ReadRequestHeaders() ([]rpcRequest, bool, Error) {
	c.decMu.Lock()
	defer c.decMu.Unlock()

	var incomingMsg json.RawMessage
	if err := c.decode(&incomingMsg); err != nil {
		return nil, false, &invalidRequestError{err.Error()}
	}

	ip := strings.Split(fmt.Sprint(c.rw.RemoteAddr()), ":")[0]
	if ip == "nil" {
		ip = "none"
	}
	var req []rpcRequest
	var ok bool
	var err Error
	isBatch := isBatch(incomingMsg)
	if isBatch {
		req, ok, err = parseBatchRequest(incomingMsg)
		if err == nil {
			log.Info("batch rpc request", "payloadSize", len(incomingMsg),
				"from", json.RawMessage(ip), "requests", len(req), "caller2", fmt.Sprintf("%+v", log.Caller(1)))
		}
	} else {
		req, ok, err = parseRequest(incomingMsg)
		if _DEBUG_RPC {
			if err == nil {
				log.Info("rpc request", "payloadSize", len(incomingMsg),
					"from", ip, // or 'pipe' or '@'
					"method", req[0].service+"_"+req[0].method,
					"caller2", fmt.Sprintf("%v", log.Caller(1)), // eg:
				)
			}
		}
	}
	return req, ok, err
}

// checkReqId returns an error when the given reqId isn't valid for RPC method calls.
// valid id's are strings, numbers or null
func checkReqId(reqId json.RawMessage) error {
	if len(reqId) == 0 {
		return fmt.Errorf("missing request id")
	}
	if _, err := strconv.ParseFloat(string(reqId), 64); err == nil {
		return nil
	}
	var str string
	if err := json.Unmarshal(reqId, &str); err == nil {
		return nil
	}
	return fmt.Errorf("invalid request id")
}

// parseRequest will parse a single request from the given RawMessage. It will return
// the parsed request, an indication if the request was a batch or an error when
// the request could not be parsed.
func ParseRequest(incomingMsg json.RawMessage) ([]rpcRequest, bool, Error) {
	return parseRequest(incomingMsg)
}
func parseRequest(incomingMsg json.RawMessage) ([]rpcRequest, bool, Error) {
	var in JsonRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	if err := checkReqId(in.Id); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}
	if _DEBUG_RPC {
		log.Info("incoming request", "method", in.Method, "payload", in.Payload)
	}
	// try keeping eth compatibility
	if strings.HasPrefix(in.Method, "eth_") {
		in.Method = "aqua_" + in.Method[4:]
	}

	// eg. getblockcount -> btc_getblockcount
	if !strings.Contains(in.Method, "_") {
		in.Method = "btc_" + in.Method
	}

	// subscribe are special, they will always use `subscribeMethod` as first param in the payload
	if strings.HasSuffix(in.Method, SubscribeMethodSuffix) {
		reqs := []rpcRequest{{id: &in.Id, isPubSub: true}}
		if len(in.Payload) > 0 {
			// first param must be subscription name
			var subscribeMethod [1]string
			if err := json.Unmarshal(in.Payload, &subscribeMethod); err != nil {
				log.Debug(fmt.Sprintf("Unable to parse subscription method: %v\n", err))
				return nil, false, &invalidRequestError{"Unable to parse subscription request"}
			}

			reqs[0].service, reqs[0].method = strings.TrimSuffix(in.Method, SubscribeMethodSuffix), subscribeMethod[0]
			reqs[0].params = in.Payload
			return reqs, false, nil
		}
		return nil, false, &invalidRequestError{"Unable to parse subscription request"}
	}

	if strings.HasSuffix(in.Method, UnsubscribeMethodSuffix) {
		return []rpcRequest{{id: &in.Id, isPubSub: true,
			method: in.Method, params: in.Payload}}, false, nil
	}

	elems := strings.Split(in.Method, ServiceMethodSeparator)
	if len(elems) != 2 {
		return nil, false, &methodNotFoundError{in.Method, ""}
	}

	// regular RPC call
	if len(in.Payload) == 0 {
		return []rpcRequest{{service: elems[0], method: elems[1], id: &in.Id}}, false, nil
	}

	return []rpcRequest{{service: elems[0], method: elems[1], id: &in.Id, params: in.Payload}}, false, nil
}

// parseBatchRequest will parse a batch request into a collection of requests from the given RawMessage, an indication
// if the request was a batch or an error when the request could not be read.
func parseBatchRequest(incomingMsg json.RawMessage) ([]rpcRequest, bool, Error) {
	var in []jsonRequest
	if err := json.Unmarshal(incomingMsg, &in); err != nil {
		return nil, false, &invalidMessageError{err.Error()}
	}

	requests := make([]rpcRequest, len(in))
	for i, r := range in {
		if err := checkReqId(r.Id); err != nil {
			return nil, false, &invalidMessageError{err.Error()}
		}

		id := &in[i].Id

		// subscribe are special, they will always use `subscriptionMethod` as first param in the payload
		if strings.HasSuffix(r.Method, SubscribeMethodSuffix) {
			requests[i] = rpcRequest{id: id, isPubSub: true}
			if len(r.Payload) > 0 {
				// first param must be subscription name
				var subscribeMethod [1]string
				if err := json.Unmarshal(r.Payload, &subscribeMethod); err != nil {
					log.Debug(fmt.Sprintf("Unable to parse subscription method: %v\n", err))
					return nil, false, &invalidRequestError{"Unable to parse subscription request"}
				}

				requests[i].service, requests[i].method = strings.TrimSuffix(r.Method, SubscribeMethodSuffix), subscribeMethod[0]
				requests[i].params = r.Payload
				continue
			}

			return nil, true, &invalidRequestError{"Unable to parse (un)subscribe request arguments"}
		}

		if strings.HasSuffix(r.Method, UnsubscribeMethodSuffix) {
			requests[i] = rpcRequest{id: id, isPubSub: true, method: r.Method, params: r.Payload}
			continue
		}

		if len(r.Payload) == 0 {
			requests[i] = rpcRequest{id: id, params: nil}
		} else {
			requests[i] = rpcRequest{id: id, params: r.Payload}
		}
		if elem := strings.Split(r.Method, ServiceMethodSeparator); len(elem) == 2 {
			requests[i].service, requests[i].method = elem[0], elem[1]
		} else {
			requests[i].err = &methodNotFoundError{r.Method, ""}
		}
	}

	return requests, true, nil
}

// ParseRequestArguments tries to parse the given params (json.RawMessage) with the given
// types. It returns the parsed values or an error when the parsing failed.
func (c *jsonCodec) ParseRequestArguments(argTypes []reflect.Type, params interface{}) ([]reflect.Value, Error) {
	if args, ok := params.(json.RawMessage); !ok {
		return nil, &invalidParamsError{"Invalid params supplied"}
	} else {
		return parsePositionalArguments(args, argTypes)
	}
}

// parsePositionalArguments tries to parse the given args to an array of values with the
// given types. It returns the parsed values or an error when the args could not be
// parsed. Missing optional arguments are returned as reflect.Zero values.
func parsePositionalArguments(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, Error) {
	// Read beginning of the args array.
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	if tok, _ := dec.Token(); tok != json.Delim('[') {
		return nil, &invalidParamsError{"non-array args"}
	}
	// Read args.
	args := make([]reflect.Value, 0, len(types))
	for i := 0; dec.More(); i++ {
		if i >= len(types) {
			return nil, &invalidParamsError{fmt.Sprintf("too many arguments, want at most %d", len(types))}
		}
		argval := reflect.New(types[i])
		if err := dec.Decode(argval.Interface()); err != nil {
			return nil, &invalidParamsError{fmt.Sprintf("invalid argument %d: %v", i, err)}
		}
		if argval.IsNil() && types[i].Kind() != reflect.Ptr {
			return nil, &invalidParamsError{fmt.Sprintf("missing value for required argument %d", i)}
		}
		args = append(args, argval.Elem())
	}
	// Read end of args array.
	if _, err := dec.Token(); err != nil {
		return nil, &invalidParamsError{err.Error()}
	}
	// Set any missing args to nil.
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Ptr {
			return nil, &invalidParamsError{fmt.Sprintf("missing value for required argument %d", i)}
		}
		args = append(args, reflect.Zero(types[i]))
	}
	return args, nil
}

// CreateResponse will create a JSON-RPC success response with the given id and reply as result.
func (c *jsonCodec) CreateResponse(id interface{}, reply interface{}) interface{} {
	if isHexNum(reflect.TypeOf(reply)) {
		return &jsonSuccessResponse{Version: JsonrpcVersion, Id: id, Result: fmt.Sprintf(`%#x`, reply)}
	}
	return &jsonSuccessResponse{Version: JsonrpcVersion, Id: id, Result: reply}
}

// CreateErrorResponse will create a JSON-RPC error response with the given id and error.
func (c *jsonCodec) CreateErrorResponse(id interface{}, err Error) interface{} {
	return &jsonErrResponse{Version: JsonrpcVersion, Id: id, Error: jsonError{Code: err.ErrorCode(), Message: err.Error()}}
}

// CreateErrorResponseWithInfo will create a JSON-RPC error response with the given id and error.
// info is optional and contains additional information about the error. When an empty string is passed it is ignored.
func (c *jsonCodec) CreateErrorResponseWithInfo(id interface{}, err Error, info interface{}) interface{} {
	return &jsonErrResponse{Version: JsonrpcVersion, Id: id,
		Error: jsonError{Code: err.ErrorCode(), Message: err.Error(), Data: info}}
}

// CreateNotification will create a JSON-RPC notification with the given subscription id and event as params.
func (c *jsonCodec) CreateNotification(subid, namespace string, event interface{}) interface{} {
	if isHexNum(reflect.TypeOf(event)) {
		return &jsonNotification{Version: JsonrpcVersion, Method: namespace + NotificationMethodSuffix,
			Params: jsonSubscription{Subscription: subid, Result: fmt.Sprintf(`%#x`, event)}}
	}

	return &jsonNotification{Version: JsonrpcVersion, Method: namespace + NotificationMethodSuffix,
		Params: jsonSubscription{Subscription: subid, Result: event}}
}

// Write message to client
func (c *jsonCodec) Write(res interface{}) error {
	c.encMu.Lock()
	defer c.encMu.Unlock()

	return c.encode(res)
}

// Close the underlying connection
func (c *jsonCodec) Close() {
	c.closer.Do(func() {
		close(c.closed)
		c.rw.Close()
	})
}

// Closed returns a channel which will be closed when Close is called
func (c *jsonCodec) Closed() <-chan interface{} {
	return c.closed
}
