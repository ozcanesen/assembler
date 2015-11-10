package main

import (
    "fmt"
    "log"
    "os"
    "bufio"
    "sync"
    "flag"
    "strconv"
    "runtime/pprof"
    "github.com/ozcanesen/assembler/fasta"
)

var kmer_size int
var kmer_counts = make(map[string]uint32)
var kmer_queue = make(chan map[string]uint32)
var wg sync.WaitGroup
 

func main() {
    kmerPtr := flag.Int("k", 27, "k-mer size")
    fastaPtr := flag.String("f", "", "fasta file")
    outputPtr := flag.String("o", "kmers.gob", "output file")
    cpuprofilePtr := flag.String("cpuprofile", "", "write cpu profile to file")
    flag.Parse()

    var fastaReader fasta.FqReader
    file, err := os.Open(*fastaPtr)
    fastaReader.Reader = bufio.NewReader(file)

    record, finished := fastaReader.Iter()
    for {
        fmt.Println(record.Seq)
        record, finished = fastaReader.Iter()
        if (finished) {
            break
        }
    }

    if *cpuprofilePtr != "" {
        f, err := os.Create(*cpuprofilePtr)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }

    kmer_size = *kmerPtr
    file, err = os.Open(*fastaPtr)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    i := 0
    go store_kmers()

    scanner := bufio.NewScanner(file)
    var sbuff []string
    for scanner.Scan() {
        if (i % 4 == 1) {
            // fasta file contains new data in each 4 lines.
            sbuff = append(sbuff, scanner.Text())
        }

        if (i % 40000 == 1) {
            wg.Add(1)
            go get_kmers(sbuff)
            sbuff = nil

        }

        i += 1
    }
    // rest
    wg.Add(1)
    go get_kmers(sbuff)
    wg.Wait()
    write_to_disk(outputPtr)

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
}

func write_to_disk(outputPtr *string) {
    if *outputPtr != "" {
        f, err := os.Create(*outputPtr)
        if err != nil {
            log.Fatal(err)
        }
        writer := bufio.NewWriter(f)
        for kmer := range kmer_counts {
            writer.WriteString(kmer + "\n" + strconv.Itoa(int(kmer_counts[kmer])) + "\n")
        }

        writer.Flush()
        f.Close()
    }
}

func store_kmers() {
    // merge local results in single map
    for local_kmer_result := range kmer_queue {
        for kmer, val := range local_kmer_result {
            kmer_counts[kmer] = kmer_counts[kmer] + val
        }
    }
}

func get_kmers(reads []string) {
    var local_kmer_counts = make(map[string]uint32)
    
    for read_index := range reads {
        read := reads[read_index]
        length := len(read)

        for i := 0; i < length - kmer_size; i++ {
            local_kmer_counts[read[i:i+kmer_size]]++
        }
    }

    kmer_queue <- local_kmer_counts
    wg.Done()
}

