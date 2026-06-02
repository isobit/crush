Evaluate scientific and engineering computations using Numbat — a statically-typed language with first-class support for physical dimensions and units. Catches dimension mismatches at parse time (e.g. adding meters to seconds is a type error). Requires `numbat` to be installed.

IMPORTANT: Always use units — never use bare numbers as a calculator. Numbat's value is dimensional analysis and type safety. If no built-in unit exists, define one with `unit Foo`.

When you get a dimension mismatch error, read the error message and fix the expression — never fall back to bare numbers. The error tells you exactly which dimensions conflict; adjust by adding or removing a factor to balance them.

Key features:
- Physical units as first-class types with automatic dimensional analysis
- Unit conversion with `->` operator
- Variables, functions, conditionals, loops
- Large prelude: SI, US customary, imperial, astronomical, atomic units, physical constants, and currencies (USD, EUR, GBP, JPY, etc.)

Examples:
```
# Unit conversion
30 km/h -> mph

# Dimensional arithmetic
let mass = 75 kg
let height = 1.82 m
let bmi = mass / height² -> kg/m²

# Physics
let v = 120 km/h
let t = 3 s
let d = v * t -> m

# Constants
let E = 0.1 kg * c²  -> MJ

# Custom units for domain-specific dimensions
unit ACU
unit request
let compute_rate = 0.12 USD / (ACU * hour)
let io_rate = 0.20 USD / (million * request)
let cost = compute_rate * 4 ACU * 1 month -> USD

# Functions
fn kinetic_energy(m: Mass, v: Velocity) -> Energy = 0.5 * m * v²
kinetic_energy(1500 kg, 60 km/h) -> kJ

# Temperature
200 °F -> °C

# Date/time
now() -> unixtime
3 weeks + 2 days -> hours
```

Note: This tool requires `numbat` to be installed on the system. If it is not available, the tool will return an error suggesting installation.
