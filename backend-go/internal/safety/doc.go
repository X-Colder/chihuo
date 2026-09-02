// Package safety contains the food-safety domain model and application
// service seam. It deliberately has no HTTP, database, or mini-program
// dependencies so those adapters can be integrated without bypassing the
// state machines or evidence-chain invariants.
package safety
