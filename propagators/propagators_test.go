// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package propagators_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagators"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceIDStr = "4bf92f3577b34da6a3ce929d0e0e4736"
	spanIDStr  = "00f067aa0ba902b7"
)

var (
	traceID = mustTraceIDFromHex(traceIDStr)
	spanID  = mustSpanIDFromHex(spanIDStr)
)

func mustTraceIDFromHex(s string) (t trace.TraceID) {
	var err error
	t, err = trace.TraceIDFromHex(s)
	if err != nil {
		panic(err)
	}
	return
}

func mustSpanIDFromHex(s string) (t trace.SpanID) {
	var err error
	t, err = trace.SpanIDFromHex(s)
	if err != nil {
		panic(err)
	}
	return
}

type outOfThinAirPropagator struct {
	t *testing.T
}

var _ otel.TextMapPropagator = outOfThinAirPropagator{}

func (p outOfThinAirPropagator) Extract(ctx context.Context, carrier otel.TextMapCarrier) context.Context {
	sc := trace.SpanContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: 0,
	}
	require.True(p.t, sc.IsValid())
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

func (outOfThinAirPropagator) Inject(context.Context, otel.TextMapCarrier) {}

func (outOfThinAirPropagator) Fields() []string {
	return nil
}

type nilCarrier struct{}

var _ otel.TextMapCarrier = nilCarrier{}

func (nilCarrier) Get(key string) string {
	return ""
}

func (nilCarrier) Set(key string, value string) {}

func TestMultiplePropagators(t *testing.T) {
	ootaProp := outOfThinAirPropagator{t: t}
	ns := nilCarrier{}
	testProps := []otel.TextMapPropagator{
		propagators.TraceContext{},
	}
	bg := context.Background()
	// sanity check of oota propagator, ensuring that it really
	// generates the valid span context out of thin air
	{
		ctx := ootaProp.Extract(bg, ns)
		sc := trace.RemoteSpanContextFromContext(ctx)
		require.True(t, sc.IsValid(), "oota prop failed sanity check")
	}
	// sanity check for real propagators, ensuring that they
	// really are not putting any valid span context into an empty
	// go context in absence of the HTTP headers.
	for _, prop := range testProps {
		ctx := prop.Extract(bg, ns)
		sc := trace.RemoteSpanContextFromContext(ctx)
		require.Falsef(t, sc.IsValid(), "%#v failed sanity check", prop)
	}
	for _, prop := range testProps {
		props := otel.NewCompositeTextMapPropagator(ootaProp, prop)
		ctx := props.Extract(bg, ns)
		sc := trace.RemoteSpanContextFromContext(ctx)
		assert.Truef(t, sc.IsValid(), "%#v clobbers span context", prop)
	}
}
