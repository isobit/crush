package tools

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func numbatAvailable() bool {
	_, err := exec.LookPath("numbat")
	return err == nil
}

func TestEvalNumbat_BasicMath(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	result, err := evalNumbat(context.Background(), "2 + 3")
	require.NoError(t, err)
	require.Contains(t, result, "5")
}

func TestEvalNumbat_UnitConversion(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	result, err := evalNumbat(context.Background(), "1 km -> m")
	require.NoError(t, err)
	require.Contains(t, result, "1000")
}

func TestEvalNumbat_DimensionalArithmetic(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	result, err := evalNumbat(context.Background(), "60 km/h * 2 h -> km")
	require.NoError(t, err)
	require.Contains(t, result, "120")
}

func TestEvalNumbat_DimensionError(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	_, err := evalNumbat(context.Background(), "1 m + 1 s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "numbat error")
}

func TestEvalNumbat_MultiLine(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	code := `let mass = 75 kg
let height = 1.82 m
mass / height^2 -> kg/m^2`

	result, err := evalNumbat(context.Background(), code)
	require.NoError(t, err)
	require.Contains(t, result, "kg/m")
}

func TestEvalNumbat_NotInstalled(t *testing.T) {
	// Override PATH to ensure numbat isn't found.
	t.Setenv("PATH", "/nonexistent")

	_, err := evalNumbat(context.Background(), "1+1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not installed")
}

func TestEvalNumbat_EmptyCode(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	// Empty input still runs numbat, which produces no output.
	result, err := evalNumbat(context.Background(), "")
	// Numbat with empty stdin might error or produce nothing.
	if err == nil {
		require.Equal(t, "(no output)", result)
	}
}

func TestEvalNumbat_PhysicalConstants(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	result, err := evalNumbat(context.Background(), "speed_of_light -> m/s")
	require.NoError(t, err)
	require.Contains(t, result, "299")
}

func TestEvalNumbat_Temperature(t *testing.T) {
	t.Parallel()
	if !numbatAvailable() {
		t.Skip("numbat not installed")
	}

	result, err := evalNumbat(context.Background(), "100 °C -> °F")
	require.NoError(t, err)
	require.Contains(t, result, "212")
}
