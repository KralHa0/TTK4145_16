Exercise 1 - Theory questions
-----------------------------

### Concepts

What is the difference between *concurrency* and *parallelism*?
> Concurrency is when you have one worker whom has many tasks to do, as in only one thing happening at a time, but many things are in progress. Parallelism is when you have multiple workers doing things at the same time. It is import to note that parallelism is here a special case of concurrency as you cannot have parallelism without it. 

What is the difference between a *race condition* and a *data race*? 
> Data race is when two or more threads access the same memory location at the same time, where at least one is a write operation and there is no sycronization. A race condition on the other hand happens when the correctnes of a program depends on the timing/ordering og concurrent operations. Task 1 in the assignment is an example og this. All data races are race conditions. Not all race conditions are data races.
 
*Very* roughly - what does a *scheduler* do, and how does it do it?
> A scheduler is responsible for deciding which task or thread runs at any given time when there are more tasks than available CPU cores. It keeps track of runnable and blocked tasks, selects which one should execute next based on a scheduling policy, and switches execution between tasks by saving and restoring their state. By rapidly switching tasks and handling blocking and wake ups, the scheduler makes concurrent programs run efficiently and appear to execute simultaneously.


### Engineering

Why would we use multiple threads? What kinds of problems do threads solve?
> We use multiple threads to allow a program to do several things at once or to make progress on one task while waiting on another. Threads help hide latency, for example when waiting for input, network, or disk operations, and they improve responsiveness by preventing one slow operation from blocking the entire program. They also allow programs to take advantage of multiple CPU cores by performing work in parallel, which can significantly improve performance for compute intensive tasks.

Some languages support "fibers" (sometimes called "green threads") or "coroutines"? What are they, and why would we rather use them over threads?
> Fibers, also called green threads or coroutines, are lightweight units of execution that are managed in user space rather than by the operating system. Unlike OS threads, they are cheaper to create, use much less memory, and can be switched between more quickly because they do not require kernel involvement. We prefer them over threads when we need to handle many concurrent tasks, such as network servers or event driven programs, because they scale better, reduce overhead, and give the language runtime more control over scheduling and resource usage.

Does creating concurrent programs make the programmer's life easier? Harder? Maybe both?
> Creating concurrent programs can make the programmer’s life both easier and harder. It is easier because concurrency lets us structure programs in a natural way, for example by separating independent tasks like input handling, computation, and output, which can improve clarity and responsiveness. At the same time, it is harder because concurrency introduces complexity such as race conditions, deadlocks, and nondeterministic behavior, which make programs more difficult to reason about, test, and debug.

What do you think is best - *shared variables* or *message passing*?
> In general, message passing is the better choice, especially in modern concurrent systems, but there are cases where shared variables are appropriate. Message passing makes programs easier to reason about because it avoids shared mutable state, which reduces data races and makes ownership and synchronization explicit. Shared variables can be more efficient and natural for low level or performance critical code, but they are harder to use correctly and more prone to subtle bugs. In practice, message passing is usually preferred for structuring concurrent programs, while shared variables with locks are reserved for situations where they are truly necessary.


