package domain

// Money is an amount in Chilean pesos (CLP). CLP has no cents, so an amount is
// always a whole number of pesos held as an int64. Money must never be a float:
// float rounding would corrupt the min-increment comparison when placing bids.
type Money int64
