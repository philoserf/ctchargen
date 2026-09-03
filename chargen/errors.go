package chargen

import "errors"

// errNotAWeapon reports a decider that named a weapon the page does not
// print for the category it was asked about.
var errNotAWeapon = errors.New("not a weapon of that category")

// errNoSuchStrategy reports a strategy name no row of POLICY.md carries.
var errNoSuchStrategy = errors.New("no such strategy")
