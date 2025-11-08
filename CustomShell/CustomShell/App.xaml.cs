using System;
using System.Threading;
using System.Windows;

namespace CustomShell
{
    public partial class App : Application
    {
        private static Mutex _mutex = null;

        protected override void OnStartup(StartupEventArgs e)
        {
            const string mutexName = "CustomShellMutex";
            bool createdNew;

            _mutex = new Mutex(true, mutexName, out createdNew);

            if (!createdNew)
            {
                MessageBox.Show("Приложение уже запущено!", "Ошибка", MessageBoxButton.OK, MessageBoxImage.Error);
                Shutdown();
                return;
            }

            base.OnStartup(e);
        }
    }
}