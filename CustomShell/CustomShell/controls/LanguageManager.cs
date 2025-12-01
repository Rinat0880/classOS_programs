using System;
using System.Runtime.InteropServices;
using System.Windows.Controls;
using System.Windows.Input;

namespace CustomShell.controls
{
    public static class LanguageManager
    {
        private const int KEYEVENTF_KEYUP = 0x0002;
        private const byte VK_LWIN = 0x5B;
        private const byte VK_SPACE = 0x20;

        [DllImport("user32.dll")]
        private static extern void keybd_event(byte bVk, byte bScan, int dwFlags, int dwExtraInfo);

        public static void Init(TextBlock langText)
        {
            langText.Text = InputLanguageManager.Current.CurrentInputLanguage.TwoLetterISOLanguageName.ToUpper();

            InputLanguageManager.Current.InputLanguageChanged += (s, e) =>
            {
                langText.Text = e.NewLanguage.TwoLetterISOLanguageName.ToUpper();
            };
        }

        public static void ShowSystemLanguagePopup()
        {
            keybd_event(VK_LWIN, 0, 0, 0);
            keybd_event(VK_SPACE, 0, 0, 0);
            keybd_event(VK_SPACE, 0, KEYEVENTF_KEYUP, 0);
            keybd_event(VK_LWIN, 0, KEYEVENTF_KEYUP, 0);
        }
    }
}
