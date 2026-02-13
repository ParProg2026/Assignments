package banker

/* Restore the state to the state before considering i. */
func grant_request(i int, request, available []int, allocation, need [][]int) {
	for j := 0; j < len(available); j++ {
		available[j] = available[j] - request[j]
		allocation[i][j] = allocation[i][j] + request[j]
		need[i][j] = need[i][j] - request[j]
	}
}

/* Restore the state to the state before considering i. */
func restore_previous_state(i int, request, available []int, allocation, need [][]int) {
	for j := 0; j < len(available); j++ {
		available[j] = available[j] + request[j]
		allocation[i][j] = allocation[i][j] - request[j]
		need[i][j] = need[i][j] + request[j]
	}
}

func have_needs_of(i int, need [][]int, free []int) (ok bool) {
	ok = true
	for j := 0; ok && j < len(free); j++ {
		ok = ok && need[i][j] <= free[j]
	}
	return
}

/*
Check whether request can be granted to i by computing a safe

	schedule.
*/
func Bankers(i int, request, available []int, allocation, need [][]int) (ok bool) {
	ok = true
	for j := 0; ok && j < len(available); j++ {
		ok = ok && request[j] <= available[j]
	}
	if !ok {
		return // process must wait
	}
	grant_request(i, request, available, allocation, need)
	if ok, _ = Safe(available, allocation, need); !ok { // process i must wait
		restore_previous_state(i, request, available, allocation, need)
	}
	return
}

/*
If the state is safe, a schedule that grants maximum resources

	to all processes is returned. Otherwise, the result is false and
	a partial schedule.
*/
func Safe(available []int, allocation, need [][]int) (ok bool, schedule []int) {
	free, done := make([]int, len(available)), make([]bool, len(need))
	copy(free, available)
	for i := 0; i < len(need); {
		if !done[i] && have_needs_of(i, need, free) {
			for j := 0; j < len(free); j++ {
				free[j] += allocation[i][j]
			}
			done[i], schedule, i = true, append(schedule, i), 0
			continue
		}
		i++
	}
	ok = true
	for i := 0; ok && i < len(done); i++ {
		ok = ok && done[i]
	}
	return
}
