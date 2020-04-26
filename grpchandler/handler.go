package grpchandler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/carvalhorr/protoc-gen-mock/stub"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"strings"
)

type MockService interface {
	Register(s *grpc.Server)
	GetSupportedMethods() []string
	GetPayloadExamples() []stub.Stub
	GetStubsValidator() stub.StubsValidator
}

// MockInterceptor intercepts the gRPC calls for the registered services return canned responses previously loaded through the REST API.
var MockHandler = func(ctx context.Context, stubsMatcher stub.StubsMatcher, fullMethod string, req interface{}, resp interface{}) (_ interface{}, err error) {
	paramsJson, err := getRequestInJSON(req)
	if err != nil {
		logError(fullMethod, paramsJson, err)
		return nil, err
	}

	stub := stubsMatcher.Match(ctx, fullMethod, paramsJson)
	if stub != nil {
		if stub.Response.Type == "error" {
			return nil, fmt.Errorf(stub.Response.Error)
		}
		resp, transformErr := transformStubToResponse(stub, resp)
		if transformErr != nil {
			logError(fullMethod, paramsJson, transformErr)
			return nil, fmt.Errorf("could not unmarshal response")
		}
		log.WithFields(log.Fields{"response": resp}).
			Infof("Found MOCK response for %s --> %s", fullMethod, paramsJson)
		return resp, nil
	}

	log.Infof("NO mock response found for %s --> %s", fullMethod, paramsJson)
	return nil, fmt.Errorf("no response found")
}

func logError(fullMethod, paramsJSON string, err error) {
	log.WithFields(log.Fields{"Error": err.Error()}).
		Errorf("Error handling request %s --> %s", fullMethod, paramsJSON)
}

func getRequestInJSON(req interface{}) (requestJSON string, err error) {
	data, marshalError := json.Marshal(req)
	if marshalError != nil {
		return "", fmt.Errorf("could not marshal the request to JSON: %w", marshalError)
	}

	requestJSON = string(data)
	return
}

func transformStubToResponse(stub *stub.Stub, returnTypeInstance interface{}) (interface{}, error) {
	err := jsonpb.Unmarshal(strings.NewReader(stub.Response.Content.String()), returnTypeInstance.(proto.Message))

	if err != nil {
		return nil, err
	}
	return returnTypeInstance, nil
}
