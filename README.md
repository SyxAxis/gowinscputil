



A utility to read the WinSCP INI file. WinSCP is one of the more popular open source SFTP clients on Windows and having used it extensively I needed a way to read entries out of the WinSCP INI file and use them in golang code especially SFTP calls.

So I wrote this very basic, rough-around-edges util that will spit out the key WinSCP INI file metadata in JSON format to the STDOUT, which can then be converted to PowerShell/.Net objects.

This not to to be used production and I do not make any warranty on it. WinSCP stores a lot of infomation in relatively easy to extract clear text, especially passwords and keys. Feel free to use parts of the code for educational purposes.



WinSCP utility, libraries and code are copyright and ownership of Martin Prikryl.

https://winscp.net/eng/index.php
https://github.com/winscp/winscp
https://github.com/martinprikryl
