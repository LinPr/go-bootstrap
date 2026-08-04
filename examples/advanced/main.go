package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	bootstrap "github.com/LinPr/go-bootstrap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func main() {
	// 使用默认配置
	config := bootstrap.DefaultConfig()
	config.ServiceName = "advanced-example"
	config.ServiceVersion = "1.0.0"

	if err := bootstrap.Initialize(config); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bootstrap.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown OpenTelemetry: %v", err)
		}
	}()

	slog.Info("Advanced example started")

	ctx := context.Background()

	// 演示嵌套的 trace spans
	if err := processOrder(ctx, "order-12345"); err != nil {
		slog.Error("Failed to process order", "error", err)
	}

	slog.Info("Advanced example completed")

	time.Sleep(2 * time.Second)
}

func processOrder(ctx context.Context, orderID string) error {
	tracer := otel.Tracer("advanced-example")
	ctx, span := tracer.Start(ctx, "process-order")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.String("order.status", "processing"),
	)

	slog.Info("Processing order", "order_id", orderID)

	// 步骤 1: 验证订单
	if err := validateOrder(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Order validation failed")
		return err
	}

	// 步骤 2: 检查库存
	if err := checkInventory(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Inventory check failed")
		return err
	}

	// 步骤 3: 处理支付
	if err := processPayment(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Payment processing failed")
		return err
	}

	// 步骤 4: 发货
	if err := shipOrder(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Shipping failed")
		return err
	}

	span.SetAttributes(attribute.String("order.status", "completed"))
	span.SetStatus(codes.Ok, "Order processed successfully")

	slog.Info("Order processed successfully", "order_id", orderID)

	return nil
}

func validateOrder(ctx context.Context, orderID string) error {
	tracer := otel.Tracer("advanced-example")
	_, span := tracer.Start(ctx, "validate-order")
	defer span.End()

	slog.Debug("Validating order", "order_id", orderID)

	// 模拟验证
	time.Sleep(50 * time.Millisecond)

	span.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.Bool("validation.passed", true),
	)

	slog.Info("Order validated", "order_id", orderID)

	return nil
}

func checkInventory(ctx context.Context, orderID string) error {
	tracer := otel.Tracer("advanced-example")
	_, span := tracer.Start(ctx, "check-inventory")
	defer span.End()

	slog.Debug("Checking inventory", "order_id", orderID)

	// 模拟数据库查询
	time.Sleep(100 * time.Millisecond)

	span.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.Int("inventory.available", 50),
		attribute.Int("inventory.required", 2),
	)

	slog.Info("Inventory checked", "order_id", orderID, "available", 50)

	return nil
}

func processPayment(ctx context.Context, orderID string) error {
	tracer := otel.Tracer("advanced-example")
	ctx, span := tracer.Start(ctx, "process-payment")
	defer span.End()

	slog.Debug("Processing payment", "order_id", orderID)

	// 模拟支付网关调用
	if err := callPaymentGateway(ctx, orderID); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.String("payment.status", "completed"),
		attribute.Float64("payment.amount", 99.99),
	)

	slog.Info("Payment processed", "order_id", orderID, "amount", 99.99)

	return nil
}

func callPaymentGateway(ctx context.Context, orderID string) error {
	tracer := otel.Tracer("advanced-example")
	_, span := tracer.Start(ctx, "call-payment-gateway")
	defer span.End()

	// 模拟外部 API 调用
	time.Sleep(200 * time.Millisecond)

	span.SetAttributes(
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "https://payment-gateway.example.com/api/charge"),
		attribute.Int("http.status_code", 200),
	)

	return nil
}

func shipOrder(ctx context.Context, orderID string) error {
	tracer := otel.Tracer("advanced-example")
	_, span := tracer.Start(ctx, "ship-order")
	defer span.End()

	slog.Debug("Shipping order", "order_id", orderID)

	// 模拟创建运单
	time.Sleep(80 * time.Millisecond)

	span.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.String("shipping.carrier", "FastShip"),
		attribute.String("shipping.tracking_number", "TRACK-"+orderID),
	)

	slog.Info("Order shipped",
		"order_id", orderID,
		"carrier", "FastShip",
		"tracking", "TRACK-"+orderID,
	)

	return nil
}
