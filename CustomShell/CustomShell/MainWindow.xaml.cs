using ManagedShell;
using ManagedShell.WindowsTasks;
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interop;
using System.Windows.Threading;
using System.Windows.Media.Imaging;
using System.Windows.Media;

namespace CustomShell
{
    public partial class MainWindow : Window
    {
        private DispatcherTimer _clockTimer;
        private Dictionary<IntPtr, Button> _windowButtons;
        private TasksService _tasksService;
        private bool _isShellInitialized = false;
        private IntPtr _activeWindowHandle = IntPtr.Zero;

        public MainWindow()
        {
            InitializeComponent();
            _windowButtons = new Dictionary<IntPtr, Button>();

            InitializeClock();
            InitializeStartMenu();
            Loaded += MainWindow_Loaded;

        }

        const int GWL_EXSTYLE = -20;
        const int WS_EX_TOOLWINDOW = 0x00000080;

        [DllImport("user32.dll")]
        static extern int GetWindowLong(IntPtr hWnd, int nIndex);

        [DllImport("user32.dll")]
        static extern int SetWindowLong(IntPtr hWnd, int nIndex, int value);

        [DllImport("user32.dll")]
        static extern IntPtr GetForegroundWindow();

        protected override void OnSourceInitialized(EventArgs e)
        {
            base.OnSourceInitialized(e);

            var hwnd = new WindowInteropHelper(this).Handle;
            int exStyle = (int)GetWindowLong(hwnd, GWL_EXSTYLE);
            SetWindowLong(hwnd, GWL_EXSTYLE, exStyle | WS_EX_TOOLWINDOW);
        }


        private void MainWindow_Loaded(object sender, RoutedEventArgs e)
        {
            HideSystemTaskbar();
            PositionAndRegisterBar();
            InitializeShell();

            // Таймер для отслеживания активного окна
            var activeWindowTimer = new DispatcherTimer
            {
                Interval = TimeSpan.FromMilliseconds(500)
            };
            activeWindowTimer.Tick += (s, args) => UpdateActiveWindow();
            activeWindowTimer.Start();

            // Приводим окно к фронту
            this.Activate();
        }

        private void UpdateActiveWindow()
        {
            IntPtr foreground = GetForegroundWindow();
            if (foreground != _activeWindowHandle)
            {
                _activeWindowHandle = foreground;
                UpdateButtonStates();
            }
        }

        private void UpdateButtonStates()
        {
            foreach (var kvp in _windowButtons)
            {
                bool isActive = kvp.Key == _activeWindowHandle;
                kvp.Value.Background = isActive
                    ? new SolidColorBrush(Color.FromArgb(80, 255, 255, 255))
                    : Brushes.Transparent;
            }
        }

        private void InitializeShell()
        {
            if (_isShellInitialized) return;

            try
            {
                if (_tasksService == null)
                    _tasksService = new TasksService();

                var initializeMethod = typeof(TasksService).GetMethod("Initialize",
                    System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);

                if (initializeMethod != null)
                {
                    var isInitializedField = typeof(TasksService).GetField("_initialized",
                        System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);

                    bool isInitialized = isInitializedField != null && (bool)isInitializedField.GetValue(_tasksService);

                    if (!isInitialized)
                        initializeMethod.Invoke(_tasksService, new object[] { true });
                }

                _tasksService.WindowActivated -= TasksService_WindowActivated;
                _tasksService.WindowActivated += TasksService_WindowActivated;

                var windowsProperty = typeof(TasksService).GetProperty("Windows",
                    System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);

                if (windowsProperty != null)
                {
                    var windows = windowsProperty.GetValue(_tasksService) as
                        System.Collections.ObjectModel.ObservableCollection<ApplicationWindow>;

                    if (windows != null)
                    {
                        foreach (var window in windows)
                        {
                            AddTaskbarButton(window);
                        }

                        windows.CollectionChanged -= Windows_CollectionChanged;
                        windows.CollectionChanged += Windows_CollectionChanged;
                    }
                }

                _isShellInitialized = true;
            }
            catch (Exception ex)
            {
                MessageBox.Show($"Ошибка инициализации Shell: {ex.Message}", "Ошибка",
                    MessageBoxButton.OK, MessageBoxImage.Error);
            }
        }



        private void TasksService_WindowActivated(object sender, WindowEventArgs e)
        {
            if (e.Window != null)
            {
                Dispatcher.Invoke(() =>
                {
                    if (!_windowButtons.ContainsKey(e.Window.Handle))
                    {
                        AddTaskbarButton(e.Window);
                    }
                    _activeWindowHandle = e.Window.Handle;
                    UpdateButtonStates();
                });
            }
        }

        private void PositionAndRegisterBar()
        {
            try
            {
                double dpiScale = GetDpiScale();
                double adjustedHeight = 52 * dpiScale;

                //IntPtr taskbar = FindWindow("Shell_TrayWnd", null);
                //ShowWindow(taskbar, SW_HIDE);

                this.Left = 0;
                this.Top = SystemParameters.PrimaryScreenHeight - (adjustedHeight / dpiScale);
                this.Width = SystemParameters.PrimaryScreenWidth;
                this.Height = adjustedHeight / dpiScale;

                this.Topmost = true;
                this.WindowStyle = WindowStyle.None;
                this.ResizeMode = ResizeMode.NoResize;
            }
            catch (Exception ex)
            {
                MessageBox.Show($"Ошибка позиционирования: {ex.Message}", "Ошибка",
                    MessageBoxButton.OK, MessageBoxImage.Warning);
            }
        }

        private double GetDpiScale()
        {
            PresentationSource source = PresentationSource.FromVisual(this);
            return source != null ? source.CompositionTarget.TransformToDevice.M11 : 1.0;
        }

        #region AppBar Implementation

        [DllImport("shell32.dll")]
        private static extern IntPtr SHAppBarMessage(int dwMessage, ref APPBARDATA pData);

        [StructLayout(LayoutKind.Sequential)]
        private struct APPBARDATA
        {
            public int cbSize;
            public IntPtr hWnd;
            public int uCallbackMessage;
            public int uEdge;
            public RECT rc;
            public IntPtr lParam;
        }

        [StructLayout(LayoutKind.Sequential)]
        private struct RECT
        {
            public int left;
            public int top;
            public int right;
            public int bottom;
        }

        private const int ABM_NEW = 0;
        private const int ABM_REMOVE = 1;
        private const int ABM_QUERYPOS = 2;
        private const int ABM_SETPOS = 3;
        private const int ABE_BOTTOM = 3;

        private void UnregisterAppBar(IntPtr handle)
        {
            Console.WriteLine("UnregisterAppBar called");
            APPBARDATA abd = new APPBARDATA();
            abd.cbSize = Marshal.SizeOf(abd);
            abd.hWnd = handle;
            SHAppBarMessage(ABM_REMOVE, ref abd);
        }

        #endregion

        #region WinAPI for Hiding Taskbar

        [DllImport("user32.dll", SetLastError = true)]
        static extern IntPtr FindWindow(string lpClassName, string lpWindowName);

        [DllImport("user32.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);

        const int SW_HIDE = 0;
        const int SW_SHOW = 5;

        void HideSystemTaskbar()
        {
            IntPtr taskbar = FindWindow("Shell_TrayWnd", null);
            ShowWindow(taskbar, SW_HIDE);
        }

        void ShowSystemTaskbar()
        {
            IntPtr taskbar = FindWindow("Shell_TrayWnd", null);
            ShowWindow(taskbar, SW_SHOW);
        }

        #endregion

        private void InitializeClock()
        {
            _clockTimer = new DispatcherTimer
            {
                Interval = TimeSpan.FromSeconds(1)
            };
            _clockTimer.Tick += (s, e) => UpdateClock();
            _clockTimer.Start();
            UpdateClock();
        }

        private void UpdateClock()
        {
            TimeText.Text = DateTime.Now.ToString("HH:mm");
            DateText.Text = DateTime.Now.ToString("dd.MM.yyyy");
        }

        private void InitializeStartMenu()
        {
            var commonApps = new[]
            {
                new { Name = "Проводник", Path = "explorer.exe" },
                new { Name = "Настройки", Path = "ms-settings:" },
                new { Name = "Блокнот", Path = "notepad.exe" },
                new { Name = "Калькулятор", Path = "calc.exe" },
                new { Name = "Paint", Path = "mspaint.exe" },
                new { Name = "Командная строка", Path = "cmd.exe" },
                new { Name = "PowerShell", Path = "powershell.exe" },
                new { Name = "Диспетчер задач", Path = "taskmgr.exe" }
            };

            foreach (var app in commonApps)
            {
                var button = new Button
                {
                    Content = app.Name,
                    Height = 40,
                    Margin = new Thickness(4),
                    HorizontalContentAlignment = HorizontalAlignment.Left,
                    Padding = new Thickness(12, 0, 0, 0),
                    Background = System.Windows.Media.Brushes.Transparent,
                    BorderThickness = new Thickness(0),
                    Foreground = System.Windows.Media.Brushes.White,
                    Tag = app.Path
                };

                button.Click += StartMenuApp_Click;
                StartMenuItems.Children.Add(button);
            }
        }

        private void Windows_CollectionChanged(object sender,
            System.Collections.Specialized.NotifyCollectionChangedEventArgs e)
        {
            Dispatcher.Invoke(() =>
            {
                if (e.NewItems != null)
                {
                    foreach (ApplicationWindow window in e.NewItems)
                    {
                        AddTaskbarButton(window);
                    }
                }

                if (e.OldItems != null)
                {
                    foreach (ApplicationWindow window in e.OldItems)
                    {
                        RemoveTaskbarButton(window);
                    }
                }
            });
        }

        private ImageSource GetWindowIcon(ApplicationWindow window)
        {
            try
            {
                IntPtr hIcon = GetClassLongPtr(window.Handle, GCLP_HICON);
                if (hIcon == IntPtr.Zero)
                    hIcon = SendMessage(window.Handle, WM_GETICON, ICON_SMALL, IntPtr.Zero);
                if (hIcon == IntPtr.Zero)
                    hIcon = SendMessage(window.Handle, WM_GETICON, ICON_BIG, IntPtr.Zero);
                if (hIcon == IntPtr.Zero)
                    hIcon = GetClassLongPtr(window.Handle, GCLP_HICONSM);

                if (hIcon != IntPtr.Zero)
                {
                    return Imaging.CreateBitmapSourceFromHIcon(
                        hIcon,
                        Int32Rect.Empty,
                        BitmapSizeOptions.FromEmptyOptions());
                }
            }
            catch { }

            return null;
        }

        private void AddTaskbarButton(ApplicationWindow window)
        {
            if (_windowButtons.ContainsKey(window.Handle))
            {
                return;
            }

            if (window.ShowInTaskbar == false)
            {
                return;
            }

            var icon = new Image
            {
                Width = 20,
                Height = 20,
                Margin = new Thickness(0, 0, 8, 0),
                Source = GetWindowIcon(window)
            };

            var textBlock = new TextBlock
            {
                Text = window.Title,
                VerticalAlignment = VerticalAlignment.Center,
                TextTrimming = TextTrimming.CharacterEllipsis
            };

            var panel = new StackPanel
            {
                Orientation = Orientation.Horizontal,
                Children = { icon, textBlock }
            };

            var button = new Button
            {
                Content = panel,
                Style = FindResource("WindowButtonStyle") as Style,
                Tag = window
            };

            button.Click += (s, e) =>
            {
                var appWindow = (s as Button)?.Tag as ApplicationWindow;
                if (appWindow != null)
                {
                    WindowHelper.BringToFront(appWindow.Handle);
                    _activeWindowHandle = appWindow.Handle;
                    UpdateButtonStates();
                }
            };

            _windowButtons[window.Handle] = button;
            TaskbarItems.Children.Add(button);

            window.PropertyChanged += (s, e) =>
            {
                if (e.PropertyName == "Title" && _windowButtons.ContainsKey(window.Handle))
                {
                    Dispatcher.Invoke(() =>
                    {
                        if (_windowButtons.TryGetValue(window.Handle, out var btn))
                        {
                            if (btn.Content is StackPanel sp && sp.Children.Count > 1 && sp.Children[1] is TextBlock tb)
                            {
                                tb.Text = window.Title;
                            }
                        }
                    });
                }
            };

            UpdateButtonStates();
        }

        #region Icon Extraction

        [DllImport("user32.dll", CharSet = CharSet.Auto)]
        static extern IntPtr SendMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);

        [DllImport("user32.dll", EntryPoint = "GetClassLong")]
        static extern uint GetClassLong32(IntPtr hWnd, int nIndex);

        [DllImport("user32.dll", EntryPoint = "GetClassLongPtr")]
        static extern IntPtr GetClassLong64(IntPtr hWnd, int nIndex);

        static IntPtr GetClassLongPtr(IntPtr hWnd, int nIndex)
        {
            if (IntPtr.Size > 4)
                return GetClassLong64(hWnd, nIndex);
            else
                return new IntPtr(GetClassLong32(hWnd, nIndex));
        }

        const int GCL_HICON = -14;
        const int GCL_HICONSM = -34;
        const int GCLP_HICON = -14;
        const int GCLP_HICONSM = -34;
        const uint WM_GETICON = 0x007F;
        static readonly IntPtr ICON_SMALL = IntPtr.Zero;
        static readonly IntPtr ICON_BIG = new IntPtr(1);

        #endregion

        private void RemoveTaskbarButton(ApplicationWindow window)
        {
            if (_windowButtons.TryGetValue(window.Handle, out var button))
            {
                TaskbarItems.Children.Remove(button);
                _windowButtons.Remove(window.Handle);
            }
        }

        private void StartButton_Click(object sender, RoutedEventArgs e)
        {
            StartMenuPopup.IsOpen = !StartMenuPopup.IsOpen;
        }

        private void StartMenuApp_Click(object sender, RoutedEventArgs e)
        {
            var button = sender as Button;
            var path = button?.Tag as string;

            if (!string.IsNullOrEmpty(path))
            {
                try
                {
                    System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo
                    {
                        FileName = path,
                        UseShellExecute = true
                    });
                    StartMenuPopup.IsOpen = false;
                }
                catch (Exception ex)
                {
                    MessageBox.Show($"Не удалось запустить приложение: {ex.Message}",
                        "Ошибка", MessageBoxButton.OK, MessageBoxImage.Error);
                }
            }
        }

        private void ClockButton_Click(object sender, RoutedEventArgs e)
        {
            try
            {
                //System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo
                //{
                //    FileName = "ms-settings:dateandtime",
                //    UseShellExecute = true
                //});
            }
            catch { }
        }

        private void LanguageButton_Click(object sender, RoutedEventArgs e)
        {
            try
            {
                System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo
                {
                    FileName = "ms-settings:regionlanguage",
                    UseShellExecute = true
                });
            }
            catch { }
        }

        private void LogoutButton_Click(object sender, RoutedEventArgs e)
        {
            if (MessageBox.Show("Выйти из системы?", "Выход",
                MessageBoxButton.YesNo, MessageBoxImage.Question) == MessageBoxResult.Yes)
            {
                ExitWindows(0);
            }
        }

        private void ShutdownButton_Click(object sender, RoutedEventArgs e)
        {
            if (MessageBox.Show("Завершить работу компьютера?", "Завершение работы",
                MessageBoxButton.YesNo, MessageBoxImage.Warning) == MessageBoxResult.Yes)
            {
                ExitWindows(1);
            }
        }

        #region Windows API для выхода/завершения

        [DllImport("user32.dll", SetLastError = true)]
        private static extern bool ExitWindowsEx(uint uFlags, uint dwReason);

        private void ExitWindows(int mode)
        {
            const uint EWX_LOGOFF = 0x00000000;
            const uint EWX_SHUTDOWN = 0x00000001;
            const uint EWX_FORCE = 0x00000004;

            uint flags = mode == 0 ? EWX_LOGOFF : EWX_SHUTDOWN;
            ExitWindowsEx(flags | EWX_FORCE, 0);
        }

        #endregion

        #region Window Helper

        private static class WindowHelper
        {
            [DllImport("user32.dll")]
            private static extern bool SetForegroundWindow(IntPtr hWnd);

            [DllImport("user32.dll")]
            private static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);

            private const int SW_RESTORE = 9;

            public static void BringToFront(IntPtr handle)
            {
                ShowWindow(handle, SW_RESTORE);
                SetForegroundWindow(handle);
            }
        }
        #endregion

        private void Window_PreviewKeyDown(object sender, KeyEventArgs e)
        {
            if (e.Key == Key.System && e.SystemKey == Key.F4)
            {
                e.Handled = true;
            }

            if (Keyboard.Modifiers == (ModifierKeys.Control | ModifierKeys.Alt) && e.Key == Key.P)
            {
                this.Close();
                e.Handled = true;
            }
        }

        protected override void OnClosing(System.ComponentModel.CancelEventArgs e)
        {
            _clockTimer?.Stop();

            var handle = new WindowInteropHelper(this).Handle;
            if (handle != IntPtr.Zero)
            {
                UnregisterAppBar(handle);
            }

            _tasksService?.Dispose();

            ShowSystemTaskbar();

            base.OnClosing(e);
        }
    }
}