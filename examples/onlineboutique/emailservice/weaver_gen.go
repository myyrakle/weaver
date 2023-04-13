package emailservice

// Code generated by "weaver generate". DO NOT EDIT.
import (
	"context"
	"errors"
	"github.com/ServiceWeaver/weaver"
	"github.com/ServiceWeaver/weaver/examples/onlineboutique/types"
	"github.com/ServiceWeaver/weaver/runtime/codegen"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"reflect"
	"time"
)

func init() {
	codegen.Register(codegen.Registration{
		Name:        "github.com/ServiceWeaver/weaver/examples/onlineboutique/emailservice/T",
		Iface:       reflect.TypeOf((*T)(nil)).Elem(),
		New:         func() any { return &impl{} },
		LocalStubFn: func(impl any, tracer trace.Tracer) any { return t_local_stub{impl: impl.(T), tracer: tracer} },
		ClientStubFn: func(stub codegen.Stub, caller string) any {
			return t_client_stub{stub: stub, sendOrderConfirmationMetrics: codegen.MethodMetricsFor(codegen.MethodLabels{Caller: caller, Component: "github.com/ServiceWeaver/weaver/examples/onlineboutique/emailservice/T", Method: "SendOrderConfirmation"})}
		},
		ServerStubFn: func(impl any, addLoad func(uint64, float64)) codegen.Server {
			return t_server_stub{impl: impl.(T), addLoad: addLoad}
		},
	})
}

// Local stub implementations.

type t_local_stub struct {
	impl   T
	tracer trace.Tracer
}

func (s t_local_stub) SendOrderConfirmation(ctx context.Context, a0 string, a1 types.Order) (err error) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		// Create a child span for this method.
		ctx, span = s.tracer.Start(ctx, "emailservice.T.SendOrderConfirmation", trace.WithSpanKind(trace.SpanKindInternal))
		defer func() {
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			span.End()
		}()
	}

	return s.impl.SendOrderConfirmation(ctx, a0, a1)
}

// Client stub implementations.

type t_client_stub struct {
	stub                         codegen.Stub
	sendOrderConfirmationMetrics *codegen.MethodMetrics
}

func (s t_client_stub) SendOrderConfirmation(ctx context.Context, a0 string, a1 types.Order) (err error) {
	// Update metrics.
	start := time.Now()
	s.sendOrderConfirmationMetrics.Count.Add(1)

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		// Create a child span for this method.
		ctx, span = s.stub.Tracer().Start(ctx, "emailservice.T.SendOrderConfirmation", trace.WithSpanKind(trace.SpanKindClient))
	}

	defer func() {
		// Catch and return any panics detected during encoding/decoding/rpc.
		if err == nil {
			err = codegen.CatchPanics(recover())
			if err != nil {
				err = errors.Join(weaver.RemoteCallError, err)
			}
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			s.sendOrderConfirmationMetrics.ErrorCount.Add(1)
		}
		span.End()

		s.sendOrderConfirmationMetrics.Latency.Put(float64(time.Since(start).Microseconds()))
	}()

	// Encode arguments.
	enc := codegen.NewEncoder()
	enc.String(a0)
	(a1).WeaverMarshal(enc)
	var shardKey uint64

	// Call the remote method.
	s.sendOrderConfirmationMetrics.BytesRequest.Put(float64(len(enc.Data())))
	var results []byte
	results, err = s.stub.Run(ctx, 0, enc.Data(), shardKey)
	if err != nil {
		err = errors.Join(weaver.RemoteCallError, err)
		return
	}
	s.sendOrderConfirmationMetrics.BytesReply.Put(float64(len(results)))

	// Decode the results.
	dec := codegen.NewDecoder(results)
	err = dec.Error()
	return
}

// Server stub implementations.

type t_server_stub struct {
	impl    T
	addLoad func(key uint64, load float64)
}

// GetStubFn implements the stub.Server interface.
func (s t_server_stub) GetStubFn(method string) func(ctx context.Context, args []byte) ([]byte, error) {
	switch method {
	case "SendOrderConfirmation":
		return s.sendOrderConfirmation
	default:
		return nil
	}
}

func (s t_server_stub) sendOrderConfirmation(ctx context.Context, args []byte) (res []byte, err error) {
	// Catch and return any panics detected during encoding/decoding/rpc.
	defer func() {
		if err == nil {
			err = codegen.CatchPanics(recover())
		}
	}()

	// Decode arguments.
	dec := codegen.NewDecoder(args)
	var a0 string
	a0 = dec.String()
	var a1 types.Order
	(&a1).WeaverUnmarshal(dec)

	// TODO(rgrandl): The deferred function above will recover from panics in the
	// user code: fix this.
	// Call the local method.
	appErr := s.impl.SendOrderConfirmation(ctx, a0, a1)

	// Encode the results.
	enc := codegen.NewEncoder()
	enc.Error(appErr)
	return enc.Data(), nil
}
