package main

// Copyright © 2015 The go.notmuch Authors. Authors can be found in the AUTHORS file.
// Licensed under the GPLv3 or later.
// See COPYING at the root of the repository for details.

import (
	"flag"
	"fmt"

	notmuch "github.com/nvedant/go.notmuch"
)

var (
	dir         = flag.String("dir", "", "Notmuch database directory")
	queryString = flag.String("query", "", "Query string")
)

func main() {
	flag.Parse()
	if *dir == "" {
		fmt.Println("Please provide a database directory.")
		flag.Usage()
		return
	}
	db, err := notmuch.Open(*dir, notmuch.DBReadOnly)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
	threads, err := db.NewQuery(*queryString).Threads()
	if err != nil {
		fmt.Println(err)
	}
	var thread *notmuch.Thread
	for threads.Next(&thread) {
		m, um := thread.Authors()
		fmt.Printf("id:%s subject:%q authors:%v\n", thread.ID(), thread.Subject(), append(m, um...))
	}
}
