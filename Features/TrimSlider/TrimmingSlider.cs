using System.Globalization;
using Spectre.Console;

namespace dis.Features.TrimSlider;

public sealed class TrimmingSlider(TimeSpan duration)
{
    private readonly TimeSpan _duration = duration > TimeSpan.Zero
        ? duration
        : throw new ArgumentException("Duration must be positive", nameof(duration));

    private readonly SliderState _state = new(duration);

    private volatile bool _cancelRequested;

    public string ShowSlider()
    {
        var result = string.Empty;
        Console.CancelKeyPress += Console_CancelKeyPress;

        try
        {
            AnsiConsole.Console.AlternateScreen(() =>
            {
                while (!_cancelRequested)
                {
                    RenderInterface();

                    ConsoleKeyInfo? key = null;
                    // Loop to wait for a key or cancellation
                    while (key is null && !_cancelRequested)
                    {
                        if (Console.KeyAvailable) key = Console.ReadKey(intercept: true);
                        else
                        {
                            // IMPORTANT: Small delay to prevent a busy-wait
                            // loop This makes the CPU usage low while waiting
                            // for input or cancellation.
                            Thread.Sleep(50);
                        }
                    }

                    // If we broke out
                    if (_cancelRequested) break; // Exit the main ShowSlider loop immediately

                    // Now 'key' is guaranteed not to be null if we reached here
                    var shouldBreak = ProcessKeyPress(key!.Value, out result);
                    if (shouldBreak) break;

                    _state.RoundPositions();
                }
            });
        }
        finally
        {
            Console.CancelKeyPress -= Console_CancelKeyPress;
        }

        return _cancelRequested ? string.Empty : result;
    }

    private void Console_CancelKeyPress(object? sender, ConsoleCancelEventArgs e)
    {
        e.Cancel = true;
        _cancelRequested = true;

        // Manually send the ANSI escape code to exit the alternate screen buffer
        AnsiConsole.Console.Write(new ControlCode("\e\u005b\u003f\u0031\u0030\u0034\u0039\u006c"));
    }

    private bool ProcessKeyPress(ConsoleKeyInfo key, out string outputResult)
    {
        outputResult = string.Empty;

        if (_state.IsTypingNumber)
        {
            HandleNumberInput(key);
            return false;
        }

        if (!HandleNavigationKey(key)) return false;
        var startPosition = _state.StartPosition.ToString(CultureInfo.InvariantCulture);
        var endPosition = _state.EndPosition.ToString(CultureInfo.InvariantCulture);
        outputResult = $"{startPosition}-{endPosition}";
        return true;
    }

    private void RenderInterface()
    {
        AnsiConsole.Cursor.Hide();
        AnsiConsole.Clear();

        DrawSlider(AnsiConsole.Console);
        AnsiConsole.Console.Write(_state.IsTypingNumber
            ? DisplayStrings.GetTimeInput(_state.NumberBuffer)
            : DisplayStrings.Controls);

        AnsiConsole.Cursor.Show();
    }

    private void HandleNumberInput(ConsoleKeyInfo key)
    {
        switch (key.Key)
        {
            case ConsoleKey.Enter when !string.IsNullOrEmpty(_state.NumberBuffer):
                ProcessTimeInput();
                break;
            case ConsoleKey.Escape:
                _state.ResetNumberInput();
                break;
            case ConsoleKey.Backspace:
                _state.HandleBackspace();
                break;
            default:
                _state.AppendToBuffer(key.KeyChar);
                break;
        }
    }

    private void ProcessTimeInput()
    {
        if (!TimeParser.TryParseTimeInput(_state.NumberBuffer, out var seconds))
            return;

        var (min, max) = _state.GetValidRange();
        if (seconds >= min && seconds <= max)
        {
            _state.UpdatePosition(seconds);
        }
        _state.ResetNumberInput();
    }

    private bool HandleNavigationKey(ConsoleKeyInfo key)
    {
        if (key.Key is ConsoleKey.Escape)
        {
            _state.CancelOperation();
            return true;
        }

        var step = (key.Modifiers & ConsoleModifiers.Shift) != 0
            ? Constants.MillisecondStep
            : Constants.SecondStep;

        return key.Key switch
        {
            ConsoleKey.D1 => _state.SelectStart(),
            ConsoleKey.D2 => _state.SelectEnd(),
            ConsoleKey.Spacebar => _state.StartTyping(),
            ConsoleKey.Enter => true,
            ConsoleKey.UpArrow => _state.AdjustValue(Constants.MinuteStep, _duration),
            ConsoleKey.DownArrow => _state.AdjustValue(-Constants.MinuteStep, _duration),
            ConsoleKey.LeftArrow => _state.AdjustValue(-step, _duration),
            ConsoleKey.RightArrow => _state.AdjustValue(step, _duration),
            _ => false
        };
    }

    private void DrawSlider(IAnsiConsole console)
    {
        var slider = string.Join("", _state.GenerateSliderCharacters(Constants.SliderWidth));

        console.MarkupLine(
            $"${Environment.NewLine}Video duration: [blue]{(int)_duration.TotalMinutes:D2}:{_duration.Seconds:D2}.{_duration.Milliseconds:D3}[/]");
        console.MarkupLine($"Selected range: [green]{_state.FormatRange()}[/]{Environment.NewLine}");
        console.MarkupLine($"Currently adjusting: [blue]{(_state.IsAdjustingStart ? "Start" : "End")}[/] position{Environment.NewLine}");
        console.MarkupLine($"0s {slider} {_duration.TotalSeconds:F2}s");
    }
}
