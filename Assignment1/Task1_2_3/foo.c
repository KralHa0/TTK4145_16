/*
#include <pthread.h>
#include <stdio.h>

int i = 0;

// Note the return type: void*
void* incrementingThreadFunction(void* arg){
    for (int j = 0; j < 1000000; j++) {
        i++;
    }
    return NULL;
}

void* decrementingThreadFunction(void* arg){
    for (int j = 0; j < 1000000; j++) {
        i--;
    }
    return NULL;
}

int main(){
    pthread_t incThread;
    pthread_t decThread;

    // Start the two threads
    pthread_create(&incThread, NULL, incrementingThreadFunction, NULL);
    pthread_create(&decThread, NULL, decrementingThreadFunction, NULL);

    // Wait for both threads to finish
    pthread_join(incThread, NULL);
    pthread_join(decThread, NULL);

    printf("The magic number is: %d\n", i);
    return 0;
}
*/
#include <pthread.h>
#include <stdio.h>

int i = 0;
pthread_mutex_t lock;

// Note the return type: void*
void* incrementingThreadFunction(void* arg){
    for (int j = 0; j < 1000000; j++) {
        pthread_mutex_lock(&lock);
        i++;
        pthread_mutex_unlock(&lock);
    }
    return NULL;
}

void* decrementingThreadFunction(void* arg){
    for (int j = 0; j < 1000000; j++) {
        pthread_mutex_lock(&lock);
        i--;
        pthread_mutex_unlock(&lock);
    }
    return NULL;
}

int main(){
    pthread_t incThread;
    pthread_t decThread;

    // Initialize mutex
    pthread_mutex_init(&lock, NULL);

    // Start threads
    pthread_create(&incThread, NULL, incrementingThreadFunction, NULL);
    pthread_create(&decThread, NULL, decrementingThreadFunction, NULL);

    // Wait for threads
    pthread_join(incThread, NULL);
    pthread_join(decThread, NULL);

    // Destroy mutex
    pthread_mutex_destroy(&lock);

    printf("The magic number is: %d\n", i);
    return 0;
}
