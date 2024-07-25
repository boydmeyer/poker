# Poker

Poker is a powerful tool for managing, rolling, and resetting dice with automated poker hand evaluation and game interaction.

## Features

- Manage multiple dice: Handle operations involving multiple dice, including adding, removing, and tracking their states.
- Roll and reset dice: Simulate rolling dice and resetting them to their initial state.
- Automatically evaluate poker hands: Determine the value of poker hands based on the rolled dice.
- Automatically evaluate tri sum: Calculate the sum of three dice and evaluate specific conditions or outcomes.

## Installation

1. **Clone the repository:**

   Open your terminal and clone the repository using the following command:

   ```bash
   git clone https://github.com/boydmeyer/poker.git
   ```

2. **Navigate to the project directory:**

   Change your working directory to the project's directory:

   ```bash
   cd Poker
   ```

3. **Build the project:**

   Use the `go build` command to build the project:

   ```bash
   go build
   ```

4. **Run the project:**

   After building, execute the project with:

   ```bash
   ./Poker
   ```

## Usage

### Setup

1. **Run the Project:**

   After executing `./Poker`, the application will start running.

2. **Initialize Dice:**

   To set up, simply doubleclick all the dices. The program will record the dice in the order they were rolled. After setup is complete, you can begin using the available commands.


### Chat Commands

- `:roll` - Rolls all dice used in poker and announces the results of their values.
- `:tri` - Rolls all dice used in the tri game and announces the total sum of the three dice.
- `:close` - Closes all dices.
- `:reset` - Clears any previously stored dice data for a fresh start.
- `:chaton` - Enables announcing the results of the dice rolls
- `:chatoff` - Disables announcing the results of the dice rolls

## Contributing

Contributions are welcome! Please submit a pull request or open an issue to discuss any changes.

## License

This project is licensed under the MIT License.

```

Feel free to customize the content according to your needs.
```
