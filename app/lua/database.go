package lua

import (
	"github.com/yuin/gopher-lua"
	"strings"
	"github.com/raggaer/castro/app/database"
	"github.com/raggaer/castro/app/util"
	"time"
)

func Query(L *lua.LState) int {
	// Get query
	query := L.Get(2)

	// Check if query is valid
	if query.Type() != lua.LTString {

		// Raise error
		L.RaiseError("Cannot get article: missing QUERY")
		return 0
	}

	// Count number of params
	n := strings.Count(query.String(), "?")

	args := []interface{}{}

	// Get all arguments matching the number of params
	for i := 0; i < n; i++ {

		// Append argument to slice
		args = append(args, L.Get(3 + i).String())
	}

	// Check if user wants to use cache
	cache := L.Get(3 + n)

	// Save cache variable
	saveToCache := false
	cacheKey := query.String()

	if cache.Type() != lua.LTNil {

		// Build cache key as the full query with arguments inside
		for _, arg := range args {

			// Replace "?" with argument
			cacheKey = strings.Replace(cacheKey, "?", arg.(string), 1)
		}

		// Try to load from cache
		q, found := util.Cache.Get(cacheKey)

		if found {

			// Push the cached lua table to stack
			L.Push(q.(*lua.LTable))

			return 1
		}

		saveToCache = true
	}

	// Execute query and get rows
	rows, err := database.DB.Raw(query.String(), args).Rows()
	if err != nil {

		// Raise error
		L.RaiseError("Cannot get article: missing QUERY")
		return 0
	}

	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {

		// Raise error
		L.RaiseError("Cannot get article: missing QUERY")
		return 0
	}
	var results [][]interface{}

	// Loop result rows
	for rows.Next() {

		// Hold all result columns
		columns := make([]interface{}, len(columnNames))

		// Hold all result pointers
		columnPointers := make([]interface{}, len(columnNames))

		// Loop result columns and assign to pointer
		for i := range columnNames {
			columnPointers[i] = &columns[i]
		}

		// Scan the column pointers
		rows.Scan(columnPointers...)

		// Append to results
		results = append(results, columns)
	}

	finalTable := QueryToTable(results, columnNames)

	// If user wants to use cache save table
	if saveToCache {
		util.Cache.Add(cacheKey, finalTable, time.Minute * 3)
	}

	// Push the converted query to the stack
	L.Push(finalTable)

	return 1
}