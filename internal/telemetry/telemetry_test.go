package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestStartOperation(t *testing.T) {
	ctx := context.Background()

	operation := StartOperation(
		ctx,
		"test.operation",
		OperationAttribute("test.operation"),
	)

	if operation == nil {
		t.Fatal("expected operation")
	}

	if operation.Context == nil {
		t.Fatal("expected operation context")
	}

	if operation.Span == nil {
		t.Fatal("expected span")
	}

	if operation.name != "test.operation" {
		t.Fatalf(
			"expected operation name %q, got %q",
			"test.operation",
			operation.name,
		)
	}

	if operation.startTime.IsZero() {
		t.Fatal("expected start time")
	}

	EndOperation(operation, nil)
}

func TestStartOperationNilContext(t *testing.T) {
	operation := StartOperation(
		nil,
		"test.operation",
	)

	if operation == nil {
		t.Fatal("expected operation")
	}

	if operation.Context == nil {
		t.Fatal("expected operation context")
	}

	EndOperation(operation, nil)
}

func TestStartOperationEmptyName(t *testing.T) {
	operation := StartOperation(
		context.Background(),
		"",
	)

	if operation == nil {
		t.Fatal("expected operation")
	}

	if operation.name != "eko.operation" {
		t.Fatalf(
			"expected default operation name %q, got %q",
			"eko.operation",
			operation.name,
		)
	}

	EndOperation(operation, nil)
}

func TestEndOperationReturnsDuration(t *testing.T) {
	operation := StartOperation(
		context.Background(),
		"test.operation",
	)

	time.Sleep(1 * time.Millisecond)

	duration := EndOperation(operation, nil)

	if duration <= 0 {
		t.Fatalf(
			"expected positive duration, got %f",
			duration,
		)
	}
}

func TestEndOperationWithError(t *testing.T) {
	operation := StartOperation(
		context.Background(),
		"test.operation",
	)

	err := context.Canceled

	duration := EndOperation(operation, err)

	if duration < 0 {
		t.Fatalf(
			"expected non-negative duration, got %f",
			duration,
		)
	}
}

func TestEndOperationNilOperation(t *testing.T) {
	duration := EndOperation(nil, nil)

	if duration != 0 {
		t.Fatalf(
			"expected zero duration for nil operation, got %f",
			duration,
		)
	}
}

func TestSetErrorNilSpan(t *testing.T) {
	SetError(nil, context.Canceled)
}

func TestSetErrorNilError(t *testing.T) {
	operation := StartOperation(
		context.Background(),
		"test.operation",
	)

	SetError(operation.Span, nil)

	EndOperation(operation, nil)
}

func TestSetAttributesNilSpan(t *testing.T) {
	SetAttributes(
		nil,
		CommandAttribute("test"),
	)
}

func TestSetAttributes(t *testing.T) {
	operation := StartOperation(
		context.Background(),
		"test.operation",
	)

	SetAttributes(
		operation.Span,
		CommandAttribute("test"),
		ProviderAttribute("test-provider"),
		ModelAttribute("test-model"),
		OperationAttribute("test.operation"),
	)

	EndOperation(operation, nil)
}