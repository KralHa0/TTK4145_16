# Task 1 Explanation

With GOMAXPROCS(1), the program prints 0.
With GOMAXPROCS(2), the program prints 100000.

This is caused by a data race on the shared variable `i`.
The operations `i++` and `i--` are not atomic. 
This means that the shared variable is incremented and decremented at the same time. 
Incrementing is a faster operation so the final sum drifts towards the 100000 when we allow two OS-threads to be used.
When only one thread is used we get the wanted answer of 0, but this just masks the problem and could still show up when using different computers or at some random time.

# Task 2 Explanation

The server handles the local variable i and no other go routines can access it. This makes it so the increment and decrement always happens in their own time and guarantees the correct result each time, but is a lot of code to wright for a very simple procedure. 

# Task 3 Explanation

The buffer approach is much smaller but requires the developer too handle the deadlock such that the program does not just crash because of the routines not closing properly. 