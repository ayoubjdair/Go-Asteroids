# Go Asteroids
An Educational Game Aimed At Teaching Novice Programming Students The Concepts Of Concurrency

Submitted to the University of Limerick in fulfilment of the requirements for the Bachelor of Computer Science in Computer Systems
Department of Science and Engineering University of Limerick

# Required Library:

Ebiten.org

# Gameplay

Shoot as many asteroids as you can! Avoid collisions!

# Player

The player controls a space ship using the arrow keys. When the game is initialised, the width and height of the ship are set to constants in the code of 50 & 80 respectively. The X & Y coordinates are also set to perfectly centre the ship in the bottom half of the screen, ready to take on the asteroids above. The width and height of the screen are also stored as constants and these are used to position the space ship.


# Levels

The user has a choice between 3 difficulty levels in the game. Level 1 generates 5 asteroids concurrently which split into a total of 10 mini asteroids (slices of original asteroid) when shot down. Levels 2 and 3 increase the amount of initial concurrent asteroid generation to 10 and 20 respectively.
