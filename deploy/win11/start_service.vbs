' Start this kdash installation silently for the current interactive user.
Option Explicit
Dim WshShell, FSO, ScriptDir, ExePath, ConfigPath, Command
Set WshShell = CreateObject("WScript.Shell")
Set FSO = CreateObject("Scripting.FileSystemObject")
ScriptDir = FSO.GetParentFolderName(WScript.ScriptFullName)
ExePath = FSO.BuildPath(ScriptDir, "kdash.exe")
ConfigPath = FSO.BuildPath(ScriptDir, "kdash.json")

If Not FSO.FileExists(ExePath) Then
    MsgBox "kdash.exe was not found: " & ExePath, vbCritical, "Kindle Dashboard"
    WScript.Quit 1
End If
If Not FSO.FileExists(ConfigPath) Then
    MsgBox "kdash.json was not found: " & ConfigPath, vbCritical, "Kindle Dashboard"
    WScript.Quit 1
End If

WshShell.CurrentDirectory = ScriptDir
Command = Chr(34) & ExePath & Chr(34) & " -config " & Chr(34) & ConfigPath & Chr(34)
On Error Resume Next
WshShell.Run Command, 0, False
If Err.Number <> 0 Then
    MsgBox "Unable to start kdash.exe.", vbCritical, "Kindle Dashboard"
    WScript.Quit Err.Number
End If
