using System;

class Program {
    static void Main() {
        string text = Console.In.ReadToEnd();
        string[] tokens = text.Split(new char[] { ' ', '\n', '\r', '\t' }, StringSplitOptions.RemoveEmptyEntries);
        if (tokens.Length >= 2) {
            int a = int.Parse(tokens[0]);
            int b = int.Parse(tokens[1]);
            Console.WriteLine(a * b);
        }
    }
}
