// Package domain holds the entities of the application, the rules they share
// across modules, and the errors those rules produce.
//
// Modules depend on this package and never on each other. Every error that
// crosses a function boundary between layers or modules is declared in
// errors.go, so a module recognizes another module's failures with errors.Is
// without importing it.
package domain
