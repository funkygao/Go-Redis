//   Copyright 2009 Joubin Houshyar
// 
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//    
//   http://www.apache.org/licenses/LICENSE-2.0
//    
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//

/*
	Package protocol provides the implementation of the Redis protocol.
*/
package protocol

import (
	"os";
	"io";
	"bufio";
	"strconv";
	"bytes";
	"strings";
	"log";
	"reflect";
	"fmt";
	"redis";
 	ARGS "reflect_utils";
)
// ----------------------------------------------------------------------------
// Wire
// ----------------------------------------------------------------------------

// protocol's special bytes
const 
(
	CR_BYTE 	byte= byte('\r');
	LF_BYTE			= byte('\n');
	SPACE_BYTE 		= byte(' ');
	ERR_BYTE 		= byte(45);
	OK_BYTE 		= byte(43);
	COUNT_BYTE 		= byte(42);
	SIZE_BYTE 		= byte(36);
	NUM_BYTE 		= byte(58);
	FALSE_BYTE 		= byte(48);
	TRUE_BYTE 		= byte(49);
)

type CtlBytes []byte;
var CRLF 		CtlBytes = make([]byte, 2);
var WHITESPACE 	CtlBytes = make([]byte, 1);

func init () {
	CRLF[0] = CR_BYTE; CRLF[1] = LF_BYTE;
	WHITESPACE [0] = SPACE_BYTE;
}

// ----------------------------------------------------------------------------
// Services
// ----------------------------------------------------------------------------

// TODO: tedious but need to check for errors on all buffer writes ..
//
func CreateRequestBytes (cmd *redis.Command, v ...) ([]byte, os.Error) {

	args := reflect.NewValue(v).(*reflect.StructValue);
	cmd_bytes := strings.Bytes(cmd.Code);
	
	buffer := bytes.NewBuffer(cmd_bytes);

	switch cmd.ReqType {

	case redis.NO_ARG:
	
	case redis.KEY:
		buffer.Write(WHITESPACE);	
		key, _ := ARGS.GetByteArrayAtIndex (args, 0);
		log.Stdout("redis.Key => ", key);
		buffer.Write(key);
		
	case 
		redis.KEY_KEY, 
		redis.KEY_NUM, 
		redis.KEY_SPEC:
		
		buffer.Write(WHITESPACE);	
		key, _ := ARGS.GetByteArrayAtIndex (args, 0);
		buffer.Write(key);
		buffer.Write(WHITESPACE);	
		key2, _ := ARGS.GetByteArrayAtIndex (args, 1);
		buffer.Write(key2);
	
	case redis.KEY_NUM_NUM:
	
		buffer.Write(WHITESPACE);	
		key, _ := ARGS.GetByteArrayAtIndex (args, 0);
		buffer.Write(key);
		buffer.Write(WHITESPACE);	
		num1, _ := ARGS.GetByteArrayAtIndex (args, 1);
		buffer.Write(num1);
		buffer.Write(WHITESPACE);	
		num2, _ := ARGS.GetByteArrayAtIndex (args, 2);
		buffer.Write(num2);
	
	case redis.KEY_VALUE:
		buffer.Write(WHITESPACE);	
		key, _ := ARGS.GetByteArrayAtIndex (args, 0);
		buffer.Write(key);
		buffer.Write(WHITESPACE);
		value, _ := ARGS.GetByteArrayAtIndex (args, 1);;
		len := fmt.Sprintf("%d", len(value));
		buffer.Write(strings.Bytes(len)); 
		buffer.Write(CRLF);
		buffer.Write(value);
	
	case 
		redis.KEY_IDX_VALUE,
		redis.KEY_KEY_VALUE:
		
	case redis.KEY_CNT_VALUE:
	
	case redis.MULTI_KEY:
	}
	
	buffer.Write(CRLF);	
	
//	log.Stdout ("** \n", buffer);

	return buffer.Bytes(), nil;
}

// Gets the response to the command.
// Any errors (whether runtime or bugs) are returned as os.Error.
// The returned response (regardless of flavor) may have (application level)
// errors as sent from Redis server.
//
func GetResponse (reader *bufio.Reader, cmd *redis.Command) (resp Response, err os.Error) {
	switch cmd.RespType {
	case redis.BOOLEAN:
	    resp, err = getBooleanResponse(reader, cmd);
	case redis.BULK:
	    resp, err = getBulkResponse (reader, cmd);
	case redis.MULTI_BULK:
	    resp, err = getMultiBulkResponse (reader, cmd);
	case redis.NUMBER:
	    resp, err = getNumberResponse (reader, cmd);
	case redis.STATUS:
	    resp, err = getStatusResponse (reader, cmd);
	case redis.STRING:
	    resp, err = getStringResponse (reader, cmd);
//	case redis.VIRTUAL:
//	    resp, err = getVirtualResponse ();
	}
	return;
}

// ----------------------------------------------------------------------------
// internal ops
// ----------------------------------------------------------------------------

func getStatusResponse (conn *bufio.Reader, cmd *redis.Command) (resp Response, e os.Error) {
	buff, error, fault := readLine(conn);
	if fault == nil {
		line := bytes.NewBuffer(buff).String();
		resp = newStatusResponse(line, error);
	}
	return resp, fault;
}

func getBooleanResponse (conn *bufio.Reader, cmd *redis.Command) (resp Response, e os.Error) {
	buff, error, fault := readLine(conn);
	if fault == nil {
		if !error {
			b := buff[1] == TRUE_BYTE;
			resp = newBooleanResponse(b, error);
		}
		else { resp = newStatusResponse(bytes.NewBuffer(buff).String(), error); }
	}
	return resp, fault;
}

func getStringResponse (conn *bufio.Reader, cmd *redis.Command) (resp Response, e os.Error) {
	buff, error, fault := readLine(conn);
	if fault == nil {
		if !error {
			buff = buff[1: len(buff)];
			str := bytes.NewBuffer(buff).String();
			resp = newStringResponse(str, error);
		}
		else { resp = newStatusResponse(bytes.NewBuffer(buff).String(), error); }
	}
	return resp, fault;
}
func getNumberResponse (conn *bufio.Reader, cmd *redis.Command)  (resp Response, e os.Error) {

	buff, error, fault := readLine(conn);
	if fault == nil {
		if !error {
			buff = buff[1: len(buff)];
			numrep := bytes.NewBuffer(buff).String();
			num, err := strconv.Atoi64(numrep);
			if err == nil {  resp = newNumberResponse(num, error);  }
			else { e = os.NewError("<BUG> Expecting a int64 number representation here: " + err.String()); }
		}
		else { resp = newStatusResponse(bytes.NewBuffer(buff).String(), error); }
	}
	return resp, fault;
}

func btoi64 (buff []byte) (num int64, e os.Error) {
	numrep := bytes.NewBuffer(buff).String();
	num, e = strconv.Atoi64(numrep);
	if e != nil {
		e = os.NewError("<BUG> Expecting a int64 number representation here: " + e.String());
	}
	return;
}
func getBulkResponse (conn *bufio.Reader, cmd *redis.Command) (Response, os.Error) {
	buf, e1 := readToCRLF(conn);
	if e1 != nil { return nil, e1;}
	
	if buf[0] == ERR_BYTE {
		return newStatusResponse(bytes.NewBuffer(buf).String(), true), nil;	
	}
	if buf[0] != SIZE_BYTE {
		return nil, os.NewError("<BUG> Expecting a SIZE_BYTE in getBulkResponse");	
	}

	num, e2 := btoi64 (buf[1: len(buf)]);
	if e2 != nil { return nil, e2; }

	log.Stderr("bulk data size: ", num);
	if num < 0 {
		return newBulkResponse (nil, false), nil;	
	}
	bulkdata, e3 := readBulkData(conn, num);
	if e3 != nil { return nil, e3; }
	
	return newBulkResponse (bulkdata, false), nil;
}

func getMultiBulkResponse (conn *bufio.Reader, cmd *redis.Command) (Response, os.Error) {
	buf, e1 := readToCRLF(conn);
	if e1 != nil { return nil, e1;}
	
	if buf[0] == ERR_BYTE {
		return newStatusResponse(bytes.NewBuffer(buf).String(), true), nil;	
	}
	if buf[0] != COUNT_BYTE {
		return nil, os.NewError("<BUG> Expecting a NUM_BYTE in getMultiBulkResponse");	
	}

	num, e2 := btoi64 (buf[1: len(buf)]);
	if e2 != nil { return nil, e2; }

	log.Stderr("multibulk data count: ", num);
	if num < 0 {
		return newMultiBulkResponse (nil, false), nil;	
	}
	multibulkdata := make([][]byte, num);
	for i:=int64(0);i<num;i++ {
		sbuf, e := readToCRLF(conn);
		if e != nil { return nil, e;}
		if sbuf[0] != SIZE_BYTE {
			return nil, os.NewError("<BUG> Expecting a SIZE_BYTE for data item in getMultiBulkResponse");	
		}
		size, e2 := btoi64 (sbuf[1: len(sbuf)]);
		if e2 != nil { return nil, e2; }
		log.Stderr("item: bulk data size: ", size);
		if size < 0 {
			multibulkdata[i] = nil;	
		}
		else {
			bulkdata, e3 := readBulkData(conn, size);
			if e3 != nil { return nil, e3; }
			multibulkdata[i] = bulkdata;
		}
	}
	return newMultiBulkResponse (multibulkdata, false), nil;
}


// ----------------------------------------------------------------------------
// Response
// ----------------------------------------------------------------------------

type Response interface {
	IsError () bool;
	GetMessage() string;
	GetBooleanValue () bool;
	GetNumberValue() int64;
	GetStringValue () string;
	GetBulkData() []byte;
	GetMultiBulkData() [][]byte;
}
type _response struct {
	isError 		bool;
	msg     		string;
	boolval 		bool;
	numval  		int64;
	stringval 		string;
	bulkdata 		[]byte;
	multibulkdata 	[][]byte;
}
func (r *_response) IsError () bool { return r.isError;}
func (r *_response) GetMessage() string {return r.msg;}
func (r *_response) GetBooleanValue () bool {return r.boolval;}
func (r *_response) GetNumberValue() int64 {return r.numval;}
func (r *_response) GetStringValue () string {return r.stringval;}
func (r *_response) GetBulkData() []byte {return r.bulkdata;}
func (r *_response) GetMultiBulkData() [][]byte {return r.multibulkdata;}
func newAndInitResponse(isError bool) (r *_response) {
	r = new(_response);
	r.isError = isError;
	r.bulkdata = nil;
	r.multibulkdata = nil;
	return;
}
func newStatusResponse (msg string, isError bool) Response {
	r := newAndInitResponse(isError);
	r.msg = msg;
	return r;
}
func newBooleanResponse (val bool, isError bool) Response {
	r := newAndInitResponse(isError);
	r.boolval = val;
	return r;
}
func newNumberResponse (val int64, isError bool) Response {
	r := newAndInitResponse(isError);
	r.numval = val;
	return r;
}
func newStringResponse (val string, isError bool) Response {
	r := newAndInitResponse(isError);
	r.stringval = val;
	return r;
}
func newBulkResponse (val []byte, isError bool) Response {
	r := newAndInitResponse(isError);
	r.bulkdata = val;
	return r;
}
func newMultiBulkResponse (val [][]byte, isError bool) Response {
	r := newAndInitResponse(isError);
	r.multibulkdata = val;
	return r;
}

// ----------------------------------------------------------------------------
// Protocol i/o
// ----------------------------------------------------------------------------

// reads all bytes upto CR-LF.  (Will eat those last two bytes)
// return the line []byte up to CR-LF
// error returned is NOT ("-ERR ...").  If there is a Redis error
// that is in the line buffer returned

func readToCRLF (reader *bufio.Reader) (buffer []byte, err os.Error) {
//	reader := bufio.NewReader(conn);
	var buf []byte;
	buf, err = reader.ReadBytes(CR_BYTE);
	if err == nil {
		var b byte;
		b, err = reader.ReadByte();
		if err != nil { return; }
		if b != LF_BYTE { 
			err = os.NewError("<BUG> Expecting a Linefeed byte here!");
		}
		log.Stderr("readToCRLF: ", buf);
		buffer = buf[0 : len(buf) - 1];
	}
	return;
}

func readLine (conn *bufio.Reader) (buf []byte, error bool, fault os.Error) {
	buf, fault = readToCRLF (conn);
	if fault == nil {
		error = buf[0] == ERR_BYTE;
	}
	return;
}

func readBulkData (conn *bufio.Reader, len int64) ([]byte, os.Error) {
	buff := make([]byte, len);

	n, e := io.ReadFull (conn, buff);
	if e != nil {
		return nil, redis.NewErrorWithCause (redis.SYSTEM_ERR, "Error while attempting read of bulkdata", e);
	}
	log.Stdout ("Read ", n, " bytes.  data: ", buff);
	
	crb, e1 := conn.ReadByte();
	if e1 != nil {
		return nil, os.NewError("Error while attempting read of bulkdata terminal CR:"+ e1.String());
	}
	if crb != CR_BYTE {
		return nil, os.NewError("<BUG> bulkdata terminal was not CR as expected");
	}
	lfb, e2 := conn.ReadByte();
	if e2 != nil {
		return nil, os.NewError("Error while attempting read of bulkdata terminal LF:"+ e2.String());
	}
	if lfb != LF_BYTE {
		return nil, os.NewError("<BUG> bulkdata terminal was not LF as expected.");
	}
	
	return buff, nil;
}
