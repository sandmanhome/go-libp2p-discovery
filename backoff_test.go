package discovery

import (
	"math/rand"
	"testing"
	"time"
)

func checkDelay(bkf BackoffStrategy, expected time.Duration, t *testing.T) {
	t.Helper()
	if calculated := bkf.Delay(); calculated != expected {
		t.Fatalf("expected %v, got %v", expected, calculated)
	}
}

func TestFixedBackoff(t *testing.T) {
	startDelay := time.Second
	delay := startDelay

	bkf := NewFixedBackoffFactory(delay)
	delay *= 2
	b1 := bkf()
	delay *= 2
	b2 := bkf()

	if b1.Delay() != startDelay || b2.Delay() != startDelay {
		t.Fatal("incorrect delay time")
	}

	if b1.Delay() != startDelay {
		t.Fatal("backoff is stateful")
	}

	if b1.Reset(); b1.Delay() != startDelay {
		t.Fatalf("Reset does something")
	}
}

func TestPolynomialBackoff(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	bkf := NewPolynomialBackoffFactory(time.Second, time.Second*33, NoJitter, time.Second, []float64{0.5, 2, 3}, rng)
	b1 := bkf()
	b2 := bkf()

	if b1.Delay() != time.Second || b2.Delay() != time.Second {
		t.Fatal("incorrect delay time")
	}

	checkDelay(b1, time.Millisecond*5500, t)
	checkDelay(b1, time.Millisecond*16500, t)
	checkDelay(b1, time.Millisecond*33000, t)
	checkDelay(b2, time.Millisecond*5500, t)

	b1.Reset()
	b1.Delay()
	checkDelay(b1, time.Millisecond*5500, t)
}

func TestExponentialBackoff(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	bkf := NewExponentialBackoffFactory(time.Millisecond*650, time.Second*7, NoJitter, time.Second, 1.5, -time.Millisecond*400, rng)
	b1 := bkf()
	b2 := bkf()

	if b1.Delay() != time.Millisecond*650 || b2.Delay() != time.Millisecond*650 {
		t.Fatal("incorrect delay time")
	}

	checkDelay(b1, time.Millisecond*1100, t)
	checkDelay(b1, time.Millisecond*1850, t)
	checkDelay(b1, time.Millisecond*2975, t)
	checkDelay(b1, time.Microsecond*4662500, t)
	checkDelay(b1, time.Second*7, t)
	checkDelay(b2, time.Millisecond*1100, t)

	b1.Reset()
	b1.Delay()
	checkDelay(b1, time.Millisecond*1100, t)
}

func minMaxJitterTest(jitter Jitter, t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	if jitter(time.Nanosecond, time.Hour*10, time.Hour*20, rng) < time.Hour*10 {
		t.Fatal("Min not working")
	}
	if jitter(time.Hour, time.Nanosecond, time.Nanosecond*10, rng) > time.Nanosecond*10 {
		t.Fatal("Max not working")
	}
}

func TestNoJitter(t *testing.T) {
	minMaxJitterTest(NoJitter, t)
	for i := 0; i < 10; i++ {
		expected := time.Second * time.Duration(i)
		if calculated := NoJitter(expected, time.Duration(0), time.Second*100, nil); calculated != expected {
			t.Fatalf("expected %v, got %v", expected, calculated)
		}
	}
}

func TestFullJitter(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	minMaxJitterTest(FullJitter, t)
	const numBuckets = 51
	const multiplier = 10
	const threshold = 20

	histogram := make([]int, numBuckets)

	for i := 0; i < (numBuckets-1)*multiplier; i++ {
		started := time.Nanosecond * 50
		calculated := FullJitter(started, 0, 100, rng)
		histogram[calculated]++
	}

	for _, count := range histogram {
		if count > threshold {
			t.Fatal("jitter is not close to evenly spread")
		}
	}

	if histogram[numBuckets-1] > 0 {
		t.Fatal("jitter increased overall time")
	}
}
