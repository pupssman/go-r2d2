// Taken from https://github.com/joaocarvalhoopen/Goertzel_algorithm/blob/master/Goertzel_algorithm.go
// due to it being non-portable =(
//
// Original notice:
//
// Author:  Joao Nuno Carvalho
// Email:   joaonunocarv@gmail.com
// Date:    2017.12.10
// License: MIT OpenSource License
//
// Description: This is a implementation in Go ( GoLang ) programming language
//              of the Goertzel algorithm. The algorithm permits to determine
// the real and imaginary component of a frequency of monitoring in a signal.
// It also permit’s to know the magnitude of the vector. In the case, that one
// want’s to monitor a small number of frequencies this algorithm is more
// efficient then the FFT (Fast Fourier Transform).
// It’s is also very important when it’s necessary to distribute the computing
// time by each sample moment of processing, like for example in a interrupt
// service routine, of each sample that is acquired by a ADC. Unlike the FFT that
// has to perform the calculations in one block. One other advantage
// is that the buffer doesn’t have to be a power of 2 like in the FFT.
// And last, the frequency to be monitoring can be any specific frequency and
// to be precise doesn’t have to be the middle of a bucket like the FFT.
// The code was a port from C to Go, with several modification to permit that
// several frequencies could be monitored at the same time. The original work
// only permitted one frequency at a time.
//
// The original code in C was made by Kevin Bank, and he has a great article
// explaining the algorithm. The Link for the article is:
//    https://www.embedded.com/design/configurable-systems/4024443/The-Goertzel-Algorithm
// The link for the original code in C is:
//    https://www.embedded.com/design/embedded/source-code/4209931/09banks-txt


package main

import (
        "math"
)


//##########################
//## Start of Goertzel code.

type FrequencyBucket struct {
	TargetFrequency float32     // 941.0	//941 Hz

	coeff  float32
	Q1     float32
	Q2     float32
	sine   float32
	cosine float32

	RealPart         float32  // Real part result
	ImagPart         float32  // Imaginary part result
	MagnitureSquared float32  // Magnitude squared result.
}

type Goertzel struct {
	SampleRate int // example 8000.0 //8kHz
	BufferSize int // Can be any size but has to be at least the doble of the highest
	                 // frequency that we whant to detect.
	freqBucket []*FrequencyBucket // The slice of frequencies we want to detect and all it's internal machinery.
	}

// Call this once, to precompute the constants.
func InitGoertzel(sampleRate int, bufferSize int, frequenciesList []float32 ) (G Goertzel) {

	// Allocate the memory for the structure int the return object.
	//G := Goertzel{}
	G.SampleRate = sampleRate
	G.BufferSize = bufferSize

	// Creates a frequency bucket for each frequency.
	for _, freq := range frequenciesList{
		fB := FrequencyBucket{}
		fB.TargetFrequency = freq
		G.freqBucket = append(G.freqBucket, &fB)
	}

    log_info("For SAMPLING_RATE = %d", G.SampleRate);
    log_info(" Buffer size = %d\n", G.BufferSize);

	// Precompute the constantes for each frequency.
	for _, freqB := range G.freqBucket {
		var floatN float32 = float32(G.BufferSize);
		var k int = int(0.5 + ((floatN * freqB.TargetFrequency) / float32(G.SampleRate)))
		var omega float32 = (2.0 * math.Pi * float32(k)) / floatN;
		freqB.sine = float32(math.Sin(float64(omega)))
		freqB.cosine = float32(math.Cos(float64(omega)))
		freqB.coeff = 2.0 * freqB.cosine;
        log_info(" and FREQUENCY = %f,\n", freqB.TargetFrequency);
        log_info("k = %d and coeff = %f\n\n", k, freqB.coeff);
		G.ResetGoertzel();
	}
	return G
}

// Call this routine before every "block" (size=N) of samples.
func (G *Goertzel) ResetGoertzel(){
	// Reset's the internal state.
	for _, freqBucket:= range G.freqBucket {
		freqBucket.Q1 = 0
		freqBucket.Q2 = 0
	}
}

// Call this routine for every sample.
func (G *Goertzel) ProcessSample(sample float32) {
	for _, fB := range G.freqBucket {
		var Q0 float32 = fB.coeff * fB.Q1 - fB.Q2 + sample
		fB.Q2 = fB.Q1
		fB.Q1 = Q0
	}
}

// Basic Goertzel
// Call this routine after every block to get the complex result.
func (G *Goertzel) CalcRealImag()  {
	for _, fB := range G.freqBucket {
		fB.RealPart = (fB.Q1 - fB.Q2 * fB.cosine)
		fB.ImagPart = (fB.Q2 * fB.sine)
	}
}

// Optimized Goertzel
// Call this after every block to get the RELATIVE magnitude squared.
func (G *Goertzel) calcMagnitudeSquared() {
	for _, fB := range G.freqBucket {
		fB.MagnitureSquared = fB.Q1 * fB.Q1 + fB.Q2 * fB.Q2 - fB.Q1 * fB.Q2 * fB.coeff;
	}
}

//## End of Goertzel code.
//#######################
