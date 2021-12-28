package tracer

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

func NewTracer(serviceName string) {
	cfg := config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 0,
		},
		Gen128Bit: true,
	}
	tracer, _, err := cfg.NewTracer()
	if err != nil {
		panic(err)
	}
	opentracing.SetGlobalTracer(tracer)
}
