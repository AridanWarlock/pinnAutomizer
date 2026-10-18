package core

// Created distinguishes a new record from a successful replay.
type Created bool

// Changed identifies the caller that applied a durable transition.
type Changed bool
