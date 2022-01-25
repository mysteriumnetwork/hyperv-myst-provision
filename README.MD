# Dual-mode VPN helper

Purpose of dual-mode:
- give a user of dVPN a possibility to become a provider in one click
- run and control Mysterium node inside VM w/o Docker and its limitations (like NAT)


## Installation
* Make sure you have a keystore directory in the users profile: press Win-R and enter %USERPROFILE%\\.mysterium
* Download [myst_vm_helper](http://https://github.com/mysteriumnetwork/hyperv-myst-provision/releases/download/latest/myst-vm-helper.exe)
* Launch the executable
* Select item 1 or 2 to enable Mysterium Node.
   Note that Wifi support is experimental.
* Wait for the UI page to open in a browser.

## Known problems & solutions
**Network connection (Wifi) becomes slow after enabling the Node.**
* Disable packets coalescing on Wifi adapter. Go to Device Manager -> Network adapters -> right-click on Wifi adapter and select 'Properties'
Select 'advanced' tab, disable packets coalescing.
* Use a hardware Wifi bridge, a wired connection or a separate Wifi adapter (turn to support for futher instructions).
 
# Feedback
## Report test results.
Please share your experience.

1. OS version and build (e.g. Windows 10 version 20H2)  _____
2. Did the Node start and open the UI page ? Yes / No ____
3. Type of network connection used: Wifi / Ethernet ____
4. Did you experience any issues with network connection during the test (e.g. slow download) ? What one ? ____
