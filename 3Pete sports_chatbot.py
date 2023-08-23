class SportsChatbot:
    def __init__(self):
        self.facts = [
            "Basketball was invented by Dr. James Naismith in 1891.",
            "The fastest recorded pitch in Major League Baseball was 105.1 mph by Aroldis Chapman.",
            "The Olympic Games were first held in 776 BCE in Olympia, Greece.",
            "Brazil has won the FIFA World Cup 5 times, more than any other country.",
            "Michael Jordan is often considered the best basketball player of all time."
        ]

    def get_fact(self):
        import random
        return random.choice(self.facts)

    def respond(self, query):
        query = query.lower()
        if "fact" in query:
            return self.get_fact()
        elif "hello" in query:
            return "Hello! How can I assist you in the world of sports today?"
        else:
            return "Sorry, I don't understand that query. You can ask for a sports fact!"

    def chat(self):
        print("Hi I'm 3Pete your ai Sports Chatbot!")
        print("Type 'exit' to end the chat.")
        while True:
            user_input = input("You: ")
            if user_input.lower() == 'exit':
                print("Bot: Goodbye!")
                break
            response = self.respond(user_input)
            print(f"Bot: {response}")


# Run the chatbot
bot = SportsChatbot()
bot.chat()
