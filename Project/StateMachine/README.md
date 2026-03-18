## This is the codebase for the state machine

StateMAchine module does (on a high abstraction level) the following:
  - Initiates the elevator (elevator moves downward, until it hits a floorsensor, if it inits not ON a floorsensor).
  - Starts pulling the elevator hardware for floorsensor, buttonpresses etc.
  - If it hears from the costfuntion (A 2D-array with Up/down =  "true", representing where the elevator should go),
    it finds the single next destination where the elevator should go.  
  - The StateMachine fsm has 3 states: 
        - Idle (When there is no work to do),
        - moving, 
        - doorOpen.
    Each state is in its own file.
  - After arriving at the floor of the current order,
    StateMachine will send a message to the OrderManager to clear the order.
  - the StateMachine has 3 timers: 
        - doortimer (how long the door should stay open),
        - watchdogtimer (to make sure the fsm-code is not stuck/deadlocked),
        - floortimer (To make sure the elevator doesnt take too long moving between floors,
          which could indicate hardware failure)
  - StateMachine takes inn the worldview of the system to set the lights of the buttons.
  - Statemachine communicates the malfunction-status of the elevator to the OrderManager
    (malfunction = watchdogtimeout OR floortimer OR obstruction OR stop-button).
    This is done so that it can be communicated to the other nodes that 
    "this elevator doesnt work, so it cant take calls right now".
  - the "Stop"-button, is considered as an "emergency-stop"-button. While it is held inn,
    the elevator stops and does nothing. When it is released, the elevator resumes what it was doing.


