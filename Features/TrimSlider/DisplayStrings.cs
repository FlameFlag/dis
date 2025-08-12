using Spectre.Console;

namespace dis.Features.TrimSlider;

public static class DisplayStrings
{
    private static readonly Dictionary<string, string> Rows = new() {
        { "[aqua]1[/] / [aqua]2[/]", "Select [aqua]start/end[/] position"},
        { "[yellow]←[/] / [yellow]→[/]", "Adjust by seconds"},
        { "[yellow]↑[/] / [yellow]↓[/]", "Adjust by minutes"},
        { "[lime]Shift[/] + [yellow]←[/] / [yellow]→[/]", "Adjust by [blue]milliseconds[/]"},
        { "[aqua]Space[/]", "Enter exact time"},
        { "[green]Enter[/]", "Confirm"},
        { "[red]Esc[/]", "Cancel"}
    };

    private static Table GetTimeInputTable(string currentInput)
    {
        var table = new Table()
            .Border(TableBorder.Rounded)
            .AddColumn(new TableColumn("Time Input").Centered().PadLeft(1).PadRight(1))
            .HideHeaders();

        var currentValue = string.IsNullOrEmpty(currentInput)
            ? "█"
            : $"[underline]{currentInput}[/]█";

        table.AddRow($"[blue]Enter time value:[/] {currentValue}");
        table.AddRow(new Text("(ss) or (mm:ss) or (mm:ss.ms)", new Style(foreground: Color.Grey)));
        table.AddRow(new Markup("[green]Enter[/] to confirm, [red]Esc[/] to cancel"));

        return table;
    }

    public static Table GetTimeInput(string currentInput) => GetTimeInputTable(currentInput);

    private static Table GetControlsTable()
    {
        var table = new Table()
            .Border(TableBorder.Rounded)
            .AddColumn(new TableColumn("Key").PadLeft(1).PadRight(1))
            .AddColumn(new TableColumn("Action").PadLeft(1).PadRight(1));

        foreach (var (left, right) in Rows) table.AddRow(left, right);

        return table;
    }

    public static Table Controls => GetControlsTable();
}
