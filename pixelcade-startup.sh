#!/bin/bash

# startup script for a dedicated Pi for Pixelcade
connected=false
usbConnected=false
ethernetConnected=false
retries=0
/home/pi/pixelcade/system/announce & #this is mDNS

sleep 10

# Check USB gadget connectivity
if [[ $(cat /sys/class/net/usb0/carrier) -eq 1 ]]; then #it would return 0 if we are not USB connected
    usbConnected=true
fi

# Connectivity check for user's specific WiFi network
export DBUS_SYSTEM_BUS_ADDRESS=unix:path=/host/run/dbus/system_bus_socket

echo "Checking WiFi connectivity..."
USER_WIFI_SSID=$(head -n 1 /home/pi/pixelcade/user/pixelcade/settings/.wifi)  # Get the user's WiFi SSID

CURRENT_WIFI_SSID=$(nmcli -t -f active,ssid dev wifi | grep '^yes' | cut -d':' -f2)
if [ "$CURRENT_WIFI_SSID" == "$USER_WIFI_SSID" ]; then
    connected=true
    echo "Connected to the user's WiFi network: $CURRENT_WIFI_SSID"
else
    echo "Not connected to the user's WiFi network."
    connected=false
fi

# Are we Ethernet connected?
if [ -e /sys/class/net/eth0 ] && [ "$(cat /sys/class/net/eth0/carrier)" = "1" ]; then
    connected=true
    ethernetConnected=true
fi

if [ "$connected" = false ]; then
    echo "Attempting to connect to WiFi..."
    sudo snap connect network-manager:nmcli
    arr_lines=()
    while IFS= read -r line; do
        arr_lines+=("$line")
    done < /home/pi/pixelcade/user/pixelcade/settings/.wifi

    SSID="${arr_lines[0]}"
    PASS="${arr_lines[1]}"

    sudo -u pi nmcli c delete "${SSID}" 2>/dev/null
    effort=$(sudo -u pi nmcli d wifi connect "${SSID}" password "${PASS}" 2>&1)
    echo "$effort" | grep "successfully activated" >/dev/null
    if [ "$?" -eq 0 ]; then
        connected=true
        echo "WiFi connected successfully to $SSID."
    else
        echo "WiFi connection failed."
    fi
fi

# Remove first-time connection marker if it exists
if [[ -f "$HOME/pixelcade/deletemeafterwificonnect.txt" ]]; then
    sudo rm "$HOME/pixelcade/deletemeafterwificonnect.txt"
fi

# Check for User USB Media
echo "Checking for User USB Media..."
"/home/pi/pixelcade/system/addUSBShare.sh" &

sudo killall -9 mplayer

SYMLINK_PATH="/home/pi/pixelcade/lcdmarquees"
1920REZ="/home/pi/pixelcade/lcdmarquees1920"

# Check if path is a symlink
if [ -L "$SYMLINK_PATH" ]; then
    # Get the target of the symlink
    TARGET=$(readlink -f "$SYMLINK_PATH")
    
    # Check if the target matches the expected path
    if [ "$TARGET" = "$1920REZ" ]; then
        echo "Symlink exists and points to the correct target"
            # Define the expected content
    expected_content="console=serial
verbosity=1
bootlogo=true
disp_mode=1080p60
fb0_width=1920
fb0_height=360
rootdev=UUID=9bcd28a0-5109-4227-8666-ea72c516f639
rootfstype=ext4
usbstoragequirks=0x2537:0x1066:u,0x2537:0x1068:u"

    # Path to orange pi zero 2 boot config file
    file_path="/boot/orangepiEnv.txt"

    # Read the current content of the file
    current_content=$(cat "$file_path")

    # Compare the current content with the expected content
    if [[ "$current_content" != "$expected_content" ]]; then
        echo "File content does not match. Making changes..."

        backup_path="${file_path}.bak"
        cp "$file_path" "$backup_path"
        echo "Backup created at $backup_path"

        # Replace the corrupted file with the clean content
        echo "$expected_content" > "$file_path"
        echo "File updated with expected content."

        # Set the correct permissions
        sudo chmod 644 "$file_path"
        echo "Permissions updated to 644."
        /home/pi/pixelcade/dtext -text="Boot config file corruption detected, fixing and please reboot Pixelcade LCD"
        sleep 5
    else
        echo "File content is already correct. No changes needed."
    fi
    else
        echo "1280 resolution detected"
        #TODO add code for 1280 checks
    fi
else
    echo "Error, Path is not a symlink"
fi

if [ "$connected" = false ] && [ "$usbConnected" = false ]; then
    /home/pi/pixelcade/gsho -platform linuxfb /home/pi/pixelcade/lcdmarquees/pixelcade-error.jpg &
elif [ "$ethernetConnected" = true ] && [ "$usbConnected" = false ]; then #Ethernet only
    /home/pi/pixelcade/dtext -symbol-overlay -symbol=/home/pi/pixelcade/lcdmarquees/ethernet.png -background=/home/pi/pixelcade/lcdmarquees/pixelcade.jpg    
elif [ "$ethernetConnected" = true ] && [ "$usbConnected" = true ]; then #Ethernet and USB
     /home/pi/pixelcade/dtext -symbol-overlay -symbol=/home/pi/pixelcade/lcdmarquees/ethernet.png -symbol2=/home/pi/pixelcade/lcdmarquees/usb.png -background=/home/pi/pixelcade/lcdmarquees/pixelcade.jpg    
elif [ "$connected" = true ] && [ "$usbConnected" = false ]; then #WiFi only
     /home/pi/pixelcade/dtext -symbol-overlay -symbol=/home/pi/pixelcade/lcdmarquees/wifi.png -background=/home/pi/pixelcade/lcdmarquees/pixelcade.jpg  
elif [ "$connected" = false ] && [ "$usbConnected" = true ]; then #USB only
      /home/pi/pixelcade/dtext -symbol-overlay -symbol=/home/pi/pixelcade/lcdmarquees/usb.png -background=/home/pi/pixelcade/lcdmarquees/pixelcade.jpg    
elif [ "$connected" = true ] && [ "$usbConnected" = true ]; then #USB and WiFi
     /home/pi/pixelcade/dtext -symbol-overlay -symbol=/home/pi/pixelcade/lcdmarquees/usb.png -symbol2=/home/pi/pixelcade/lcdmarquees/wifi.png -background=/home/pi/pixelcade/lcdmarquees/pixelcade.jpg
else
    /home/pi/pixelcade/gsho -platform linuxfb /home/pi/pixelcade/lcdmarquees/pixelcade-error.jpg &
fi
