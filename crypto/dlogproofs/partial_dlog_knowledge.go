package dlogproofs

import (
	"github.com/xlab-si/emmy/crypto/common"
	"github.com/xlab-si/emmy/crypto/dlog"
	"github.com/xlab-si/emmy/types"
	"math/big"
)

func ProvePartialDLogKnowledge(dlog *dlog.ZpDLog, secret1, a1, a2, b2 *big.Int) bool {
	prover := NewPartialDLogProver(dlog)
	verifier := NewPartialDLogVerifier(dlog)

	b1, _ := prover.DLog.Exponentiate(a1, secret1)
	triple1, triple2 := prover.GetProofRandomData(secret1, a1, b1, a2, b2)

	verifier.SetProofRandomData(triple1, triple2)
	challenge := verifier.GetChallenge()

	c1, z1, c2, z2 := prover.GetProofData(challenge)
	verified := verifier.Verify(c1, z1, c2, z2)
	return verified
}

// Proving that it knows either secret1 such that a1^secret1 = b1 (mod p1) or
//  secret2 such that a2^secret2 = b2 (mod p2).
type PartialDLogProver struct {
	DLog    *dlog.ZpDLog
	secret1 *big.Int
	a1      *big.Int
	a2      *big.Int
	r1      *big.Int
	c2      *big.Int
	z2      *big.Int
	ord     int
}

func NewPartialDLogProver(dlog *dlog.ZpDLog) *PartialDLogProver {
	return &PartialDLogProver{
		DLog: dlog,
	}
}

func (prover *PartialDLogProver) GetProofRandomData(secret1, a1, b1, a2,
	b2 *big.Int) (*types.Triple, *types.Triple) {
	prover.a1 = a1
	prover.a2 = a2
	prover.secret1 = secret1
	r1 := common.GetRandomInt(prover.DLog.GetOrderOfSubgroup())
	c2 := common.GetRandomInt(prover.DLog.GetOrderOfSubgroup())
	z2 := common.GetRandomInt(prover.DLog.GetOrderOfSubgroup())
	prover.r1 = r1
	prover.c2 = c2
	prover.z2 = z2
	x1, _ := prover.DLog.Exponentiate(a1, r1)
	x2, _ := prover.DLog.Exponentiate(a2, z2)
	b2ToC2, _ := prover.DLog.Exponentiate(b2, c2)
	b2ToC2Inv := prover.DLog.Inverse(b2ToC2)
	x2, _ = prover.DLog.Multiply(x2, b2ToC2Inv)

	// we need to make sure that the order does not reveal which secret we do know:
	ord := common.GetRandomInt(big.NewInt(2))
	triple1 := types.NewTriple(x1, a1, b1)
	triple2 := types.NewTriple(x2, a2, b2)

	if ord.Cmp(big.NewInt(0)) == 0 {
		prover.ord = 0
		return triple1, triple2
	} else {
		prover.ord = 1
		return triple2, triple1
	}
}

func (prover *PartialDLogProver) GetProofData(challenge *big.Int) (*big.Int, *big.Int,
	*big.Int, *big.Int) {
	c1 := new(big.Int).Xor(prover.c2, challenge)

	z1 := new(big.Int)
	z1.Mul(c1, prover.secret1)
	z1.Add(z1, prover.r1)
	z1.Mod(z1, prover.DLog.GetOrderOfSubgroup())

	if prover.ord == 0 {
		return c1, z1, prover.c2, prover.z2
	} else {
		return prover.c2, prover.z2, c1, z1
	}
}

type PartialDLogVerifier struct {
	DLog      *dlog.ZpDLog
	triple1   *types.Triple // contains x1, a1, b1
	triple2   *types.Triple // contains x2, a2, b2
	challenge *big.Int
}

func NewPartialDLogVerifier(dlog *dlog.ZpDLog) *PartialDLogVerifier {
	return &PartialDLogVerifier{
		DLog: dlog,
	}
}

func (verifier *PartialDLogVerifier) SetProofRandomData(triple1, triple2 *types.Triple) {
	verifier.triple1 = triple1
	verifier.triple2 = triple2
}

func (verifier *PartialDLogVerifier) GetChallenge() *big.Int {
	challenge := common.GetRandomInt(verifier.DLog.GetOrderOfSubgroup())
	verifier.challenge = challenge
	return challenge
}

func (verifier *PartialDLogVerifier) verifyTriple(triple *types.Triple,
	challenge, z *big.Int) bool {
	left, _ := verifier.DLog.Exponentiate(triple.B, z)       // (a, z)
	r1, _ := verifier.DLog.Exponentiate(triple.C, challenge) // (b, challenge)
	right, _ := verifier.DLog.Multiply(r1, triple.A)         // (r1, x1)

	return left.Cmp(right) == 0
}

func (verifier *PartialDLogVerifier) Verify(c1, z1, c2, z2 *big.Int) bool {
	c := new(big.Int).Xor(c1, c2)
	if c.Cmp(verifier.challenge) != 0 {
		return false
	}

	verified1 := verifier.verifyTriple(verifier.triple1, c1, z1)
	verified2 := verifier.verifyTriple(verifier.triple2, c2, z2)
	return verified1 && verified2
}
