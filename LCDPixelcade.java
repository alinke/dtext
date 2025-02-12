package org.onebeartoe.web.enabled.pixel.controllers;
//LCD Fork
import javax.swing.*;
import me.xdrop.fuzzywuzzy.FuzzySearch;
import java.awt.*;
import java.io.File;
import java.io.FileFilter;
import java.io.FileInputStream;
import java.io.IOException;
import java.lang.reflect.Array;
import java.nio.file.FileSystems;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.*;
import java.util.List;
import java.util.concurrent.CancellationException;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.logging.Level;
import java.util.logging.Logger;
import java.util.regex.Pattern;
import java.util.regex.Matcher;

import org.apache.commons.io.IOCase;
import org.apache.commons.io.comparator.LastModifiedFileComparator;
import org.apache.commons.io.filefilter.WildcardFileFilter;
import org.apache.commons.lang.WordUtils;
import org.onebeartoe.pixel.LogMe;
import org.onebeartoe.web.enabled.pixel.DotMatrixSubDisplay;
import org.onebeartoe.web.enabled.pixel.OLEDSubDisplay;
import org.onebeartoe.web.enabled.pixel.WebEnabledPixel;

import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import static org.onebeartoe.web.enabled.pixel.WebEnabledPixel.setLCDFont;

public class LCDPixelcade {

    private static boolean isALU = System.getenv("PATH").contains("pixelcade/jre11/bin");
    public static String pixelHome = isALU ? "/opt/pixelcade/": "/home/pi/pixelcade/";
    //public static String pixelHome = "/Users/al/pixelcade/"; /for al's mac testing
    private static final String userContent = "/home/pi/pixelcade/user/pixelcade/";
    private static final String usbContent = "/media/PixelUSB/";
    //private static String pixelHome = WebEnabledPixel.getHome();
    private static String sep = File.separator;
    private static String fontPath = pixelHome + "fonts/";
    private static int loops = 0;
    private static String jarPath = new File(LCDPixelcade.class.getProtectionDomain().getCodeSource().getLocation().getPath()).getAbsolutePath();
    private static String wrapperHome = jarPath.substring(0, jarPath.lastIndexOf(File.separator)) + File.separator;
    private static String fontColor = "purple";
    //private static String DEFAULT_COMMAND = "omxplayer --aspect-mode stretch --no-osd " + pixelHome + "mp4marquees/default-pixelcade.mp4 --loop & "; //wrapperHome + "gsho -platform linuxfb " + pixelHome + "lcdmarquees/pixelcade.png";
    private static String DEFAULT_COMMAND = "mplayer -vo fbdev:/dev/fb0 \"" + pixelHome + "mp4marquees/default-pixelcade.mp4 "; //wrapperHome + "gsho -platform linuxfb " + pixelHome + "lcdmarquees/pixelcade.png";
    //private static final String MP4_COMMAND = "omxplayer --aspect-mode stretch --no-osd \"" + pixelHome + "mp4marquees/${named}.mp4\" & " + wrapperHome + "gsho -platform linuxfb \"" + pixelHome + "lcdmarquees/${named}.jpg\" ";
    private static final String MP4_COMMAND = "sudo mplayer -vo fbdev:/dev/fb0 \"${location}mp4marquees/${named}.mp4\" ; " + wrapperHome + "gsho -platform linuxfb \"${marqueeHome}lcdmarquees/${longNamed}.jpg\" ";
    private static final String MP4ONLY_COMMAND = "sudo mplayer -vo fbdev:/dev/fb0 \"${location}mp4marquees/${named}.mp4\"";
    private static final String MP4LOOP_COMMAND = "sudo mplayer -loop 0 -vo fbdev:/dev/fb0 \"${location}mp4marquees/${named}.mp4\"";
    //private static final String MP4_COMMAND = "sudo mplayer -vo fbdev:/dev/fb0 \"${location}mp4marquees/${named}.mp4\" ; " + wrapperHome + "gsho -platform linuxfb \"${marqueeHome}/${longNamed}.jpg\" ";
    private static final String JPG_COMMAND = wrapperHome + "gsho -platform linuxfb \"" + pixelHome + "lcdmarquees/${named}.jpg\"";
    private static String DTXT_COMMAND = wrapperHome + "dtext \"${txt}\" \"${fontpath}\" \"${color}\"";
    private static String PNG_COMMAND = wrapperHome + "gsho -platform linuxfb  \""+ pixelHome + "lcdmarquees/${named}.png\" ";
    //private static String GIF_COMMAND = wrapperHome + "gsho  -platform linuxfb \"${location}${system}/${named}.gif\"";  //gifs were causing issues and now we have more artwork so no need for gifs anymore
    private static String TXT_COMMAND = wrapperHome + "skrola -platform linuxfb \"${txt}\" \"${fontpath}\" \"${color}\" ${speed}";
    private static final String SLIDESHOW = "sudo fbi " + pixelHome + "lcdmarquees/* -T 1 -d /dev/fb0 -t 2 --noverbose --nocomments --fixwidth -a";
    //private static final String RESET_COMMAND = "pkill -9 omxplayer; sudo killall -9 fbi;killall -9 gsho; killall -9 skrola;";
    private static  String RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho; killall -9 skrola;";
    private static final String MARQUEE_PATH = pixelHome + "lcdmarquees/";
    private static final String ENGINE_PATH = wrapperHome + "/gsho";
    static String NOT_FOUND = pixelHome + "lcdmarquees/" + "pixelcade.jpg";
    public static String theCommand = DEFAULT_COMMAND;
    public static String currentMessage = "Welcome and Game On!";
    public static String gifSystem = "";
    public static boolean isWindows = System.getProperty("os.name").toLowerCase().startsWith("windows");
    public static boolean  doGif = false;
    public static boolean  playVid = false;
    public static boolean  isScrolling = false;
    public static String delayedSystem = "";
    public static String delayedName = "";
    private static final int ENDOFQ = 99999;//99999 is a loop value used for the Pixelcade Q indicating that this Q item should remain and not blank out when it's the last Q item
    private static final long MILLISEC = 1000;
    private static final long LOOPLEN = (long) (4 * MILLISEC); //a loop in the newWorld is 4 seconds long
    private static final double TIMEFACTOR = 3.15; //should this be variable?
    private static long loopDelay = 0;
    private static long lastCall = 0;
    private static File dir;
    private static File[] fileList = new File[]{};
    private static Process process;
    public static OLEDSubDisplay miniScreenI2C = new OLEDSubDisplay();
    private static DotMatrixSubDisplay pCadeMicro = new DotMatrixSubDisplay();
    static ArrayList<String> cleanList = new ArrayList<String>();

    static String videoLocation = "";
    static String bestGuess = "";
    static int highestConfidence = 0;

    static String videoName = "";
    private static HashMap<String,ArrayList<String>> knownSets = new HashMap<>();

    private static AppAPIObjects.TimeLine attractModeTimeLine = new AppAPIObjects.TimeLine();
    private static AppAPIObjects.SlideshowPlayer slideshowPlayer = new AppAPIObjects.SlideshowPlayer();

    static Random ran = new Random();
    static boolean hasMicoDots = WebEnabledPixel.arduino1MatrixConnected;
    static boolean hasOLED = true;
    
    private static String imageMarqueesOnly_ = WebEnabledPixel.getImageMarqueesOnly();
    private static boolean ConsoleFound = false;
    private static boolean imageMarqueeFound = false;
   //private static boolean  videoMarqueeFound = false;
    
    private static String T;
    
    private static long startTime;
    private static long endTime;
    private static long duration;
    private static String AchievementBackgroundFileName;
    private static String GameLaunchBackgroundFileName;
    private static String TextBackgroundFileName;
    private static String lcdMessage;
    private static String HighScores_ = "";
    
   // private static volatile boolean NowPlayingisRunning = false;
    //private static AtomicBoolean NowPlayingisRunning = new AtomicBoolean(false);
    //private static final Object NowPlayingLock = new Object(); //used to pause when we're showing dtext but able to interrupt it
    //private static Thread NowPlayingsleepThread; 
    
    
   // private static boolean condition = false;
    
   // private static Process externalProcess;
   // private static volatile boolean stopWaiting = false;
    //private static Thread waitingThread;
    //private static ScheduledExecutorService scheduler;
    
   //  private static final Object lock = new Object();
   // private static volatile boolean NowPlayingisRunning = true;
    private static Thread waitingThread;
    private static Thread pauseThread;
    
    private static final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(1);
   // private static AtomicBoolean NowPlayingisRunning = new AtomicBoolean(false);
    //private static final AtomicBoolean nowPlayingShowing = new AtomicBoolean(false); //moved to the main class
    private static CompletableFuture<Void> future;
    private static ExecutorService executorService = Executors.newSingleThreadExecutor();
    private static Integer NowPlayingShowTime = 5;  //5 seconds
    private static Integer HighScoreAchievementsPlayingShowTime = 10;  //to do make this configurable via API
    private static Integer dTextShowTime = 5;  //5 seconds
    private static Boolean highScoreDotFlag = false;
    private static Boolean MAMEOutputVideo = true;
    private static String DotMessage = "";
   
    
      
    public LCDPixelcade() {
        
        //to do fix
        //pixelcade dot scrolling dies after launch
        //bbh and INFO:[INTERNAL SYSTEM] Found /home/pi/pixelcade/mp4marquees/taito_type_x-Contra - Evolution Revolution.mp4
        //lcdfinder
        //set up a player for AttractMode
        slideshowPlayer = new AppAPIObjects.SlideshowPlayer();
        //attractModeTimeLine =

        if (fileList.length == 0) {

            String patternString = ".*-.*.*.jpg";

            Pattern pattern = Pattern.compile(patternString, Pattern.CASE_INSENSITIVE);

            dir = new File(pixelHome + "lcdmarquees/");
            FileFilter fileFilter = new WildcardFileFilter("*.JPG", IOCase.INSENSITIVE);  // For taking both .JPG and .jpg files (useful in *nix env)

            System.out.println("Building Fuzzy SearchList...");
            System.out.println(pixelHome + "lcdmarquees/");
            fileList = dir.listFiles(fileFilter);

            if (fileList.length > 0) {

                for (File file: fileList
                     ) {
                    if (!file.getName().contains(".undo")){

                        Matcher matcher = pattern.matcher(file.getName());
                        if ( matcher.matches()) cleanList.add(file.getName());
                    }
                }
                System.out.println( cleanList.size() +  " System Entries");
            }

        }
    }
    public static void main(String[] args) {

        String shell = "bash";

        boolean haveFBI = new File(ENGINE_PATH).exists();
        //boolean haveExtraDisplay = new File("/dev/fb1").exists();

        if (!haveFBI && WebEnabledPixel.isUnix()) {
            System.out.print("Image engine failure.\n");
        }
        if (args.length > -1) {
            try {

                if(args.length == 1)
                    runCommand(args[args.length - 1],false,0);
                else
                    displayImage(args[0],args[1],Boolean.parseBoolean(args[2]),Boolean.parseBoolean(args[3]),args[4]);
            } catch (IOException e) {
                e.printStackTrace();
            }
        } else {
            try {
                runCommand(null, false, 0);
            } catch (IOException e) {
                e.printStackTrace();
            }
        }
    }

    public static void setAttractModeTimeLine(AppAPIObjects.TimeLine attractModeTimeLine) {
        LCDPixelcade.attractModeTimeLine = attractModeTimeLine;
        System.out.println("LCD AttractMode Playlist set");
        slideshowPlayer.setTimeline(attractModeTimeLine);
    }

    public static boolean attractModeReady () {

        if (attractModeTimeLine == null || slideshowPlayer.getTimeline() == null) return false;
        return true;
    }

    public static void delayedImage(String system, String named) {
        delayedName = named;
        delayedSystem = system;
    }

    public static void startPlayer() {
        slideshowPlayer.start();
        System.out.println("LCD AttractMode Starting");
    }

    public static void stopAttractMode() {
        slideshowPlayer.stop();
        System.out.println("LCD AttractMode Stopping");
    }

    public static AppAPIObjects.SlideshowPlayer getSlideshowPlayer() {
        return slideshowPlayer;
    }

    public void setLCDFont(Font font, String fontFilename) {
       // if(!isWindows) {
            this.fontPath = fontFilename;
            System.out.print("fontPath: " + fontFilename +"\n");
            return;
       // }
    }

    public void setAltText(String text) {  //this comes from arcade handler
        this.currentMessage = text;
       // System.out.print("AltMessage set\n");
        if (!isScrolling && !ScrollingTextHttpHander.getisSubDisplayScrolling() && !text.toLowerCase().contains("dummy") && !text.toLowerCase().contains("slow")  && !text.toLowerCase().contains("exit")) {  //dont' show this if we are already scrolling on either LCD or sub display (high score for example)
            System.out.print("Pixelcade Dot Message Set\n");
            
            if (highScoreDotFlag) { //this means we have a high score scrolling and we don't want to overwrite it with a game marquee right now
                currentMessage = DotMessage;
                highScoreDotFlag = false;
            }
            
            if (currentMessage.length() < 8) {
                currentMessage = String.format("%-8s", currentMessage);
            }
            pCadeMicro.scrollText(currentMessage, 10000); //was originally 100 but then we'd stop scrolling after 100 which was not good
        }
    }

    public void setNumLoops(int loops){
        this.loops = loops;
        System.out.print("Loops set\n");
    }

    static public void displayImage(String named, String system, boolean videoOKParam, boolean marqueeOverlay, String overlayText) throws IOException {
        
      
       // startTime = System.currentTimeMillis();

   
        if (slideshowPlayer.isPlaying() || isScrolling) { //TO DO come back to this
            System.out.println("Timeline or Scrolling must be stopped first. Not Showing");
            return;
        }
         
        String marqueePath = NOT_FOUND;
        String OVERRIDE = DEFAULT_COMMAND;
        String oldNamed = named;
        String customContent = pixelHome;
        long thisCall = new Date().getTime();
        imageMarqueeFound = false;
        ConsoleFound = false;
        playVid = false;
        MAMEOutputVideo = false;
        system = system.toLowerCase(); //on Pixelcade LCD, all systems are lower case, some front ends may be sending it in upper case
        String ImagePathLowerCase =  "";
        String ImagePathCaseSensitive = "";
        String VideoPathLowerCase =  "";
        String VideoPathCaseSensitive = "";
        
//        try {
//            Thread.sleep(250);
//        } catch (InterruptedException e) {
//            e.printStackTrace();
//        }
//
//        if(thisCall - lastCall < 250){
//            System.out.println("Rejecting too-soon call: is the caller DoSing? :)");
//            return;
//        }
        lastCall = thisCall;
        videoName = named;
        
        // stopAsyncLoop();
        
        //LogMe.aLogger.info("\n"); //add a space to make the log easier to read for the user

        if (system.equals("cox")) system = "mame"; //i think this is needed for ALU
        
        //logic is we look for a console first and then a marquee, if not marquee is found, then we take the console

        // System.out.print("went here 2 display image: " + system + " " + named ); 

        //if (new File(String.format("%slcdmarquees/console/default-%s.jpg", pixelHome, system)).exists()) {
        if (new File(String.format("%slcdmarquees/default-%s.jpg", pixelHome, system)).exists()) { //as of v4.5, switching to root and no longer used /lcdmarquees/console
            ConsoleFound = true;
            //OVERRIDE = wrapperHome + "gsho -platform linuxfb \"" + pixelHome + "lcdmarquees/console/default-" + system + ".jpg\"";
            OVERRIDE = wrapperHome + "gsho -platform linuxfb \"" + pixelHome + "lcdmarquees/default-" + system + ".jpg\"";
            customContent = pixelHome;
            marqueePath = String.format("%slcdmarquees/default-%s.jpg",customContent, system);
            System.out.println("[INTERNAL] Found: JPG Console default");
            LogMe.aLogger.info("[INTERNAL] Found: JPG Console default");
        }
        //Override with user version, if present
        if (new File(String.format("%slcdmarquees/default-%s.jpg", userContent, system)).exists()) {
            ConsoleFound = true;
            OVERRIDE = wrapperHome + "gsho -platform linuxfb \"" + userContent + "lcdmarquees/default-" + system + ".jpg\"";
            customContent = userContent;
            marqueePath = String.format("%slcdmarquees/default-%s.jpg",customContent, system);
            System.out.println("[INTERNAL USER CONTENT] Found: JPG Console default");
            LogMe.aLogger.info("[INTERNAL USER CONTENT] Found: JPG Console default");
        }
        //Supercede with usbmedia, if present, this is a bug if the user adds a custom console but not a custom marquee
        if (new File(String.format("%slcdmarquees/default-%s.jpg", usbContent, system)).exists()) {
            ConsoleFound = true;
            //OVERRIDE = wrapperHome + "gsho -platform linuxfb \"" + userContent + "lcdmarquees/console/default-" + system + ".jpg\"";
            OVERRIDE = wrapperHome + "gsho -platform linuxfb \"" + usbContent + "lcdmarquees/default-" + system + ".jpg\"";
            customContent = usbContent;
            marqueePath = String.format("%slcdmarquees/default-%s.jpg",customContent, system);
            System.out.println("[USB CONTENT] Found: JPG Console default");
            LogMe.aLogger.info("[USB CONTENT] Found: JPG Console default");
        }
        //Change: Look for {system}-{named} first/consoles mitigation; "{system}-" as "path/folder"
        //******* Look for marquees now ***********
        
        LogMe.aLogger.info("[TARGET FILE NAME] " + String.format("%s-%s.jpg", system, named));
        // new File(String.format("%slcdmarquees/%s\\-%s.jpg",pixelHome, system,named)).exists()
        System.out.println("[USB CONTENT] Looking for USB Content @ " + String.format("%slcdmarquees/%s-%s.jpg", usbContent, system, named));
        //LogMe.aLogger.info("[USB CONTENT] Looking for USB Content @ " + String.format("%slcdmarquees/%s-%s.jpg", usbContent, system, named));
        // now let's look for the game marquee on USB first, if not there then look in internal
        
        if (new File(usbContent + "lcdmarquees/" + system + "-" + named + ".jpg").exists()) {
            imageMarqueeFound = true;
            System.out.println("[USB CONTENT] FOUND " + String.format("%s-%s", system, named));
            LogMe.aLogger.info("[USB CONTENT] FOUND " + String.format("lcdmarquees/%s-%s.jpg", system, named));
            System.out.println("[USB CONTENT] switching {named} to " + String.format("%s-%s", system, named));
            named = String.format("%s-%s", system, named);
            System.out.println("[USB CONTENT] named is now " + named);
            OVERRIDE = wrapperHome + "gsho  -platform linuxfb \"" + usbContent + "lcdmarquees/" + named + ".jpg\"";
            customContent = usbContent;
            //marqueePath = usbContent + "lcdmarquees/" + system + "-" + named + ".jpg";
            marqueePath = usbContent + "lcdmarquees/" + named + ".jpg";
        } 
        else if (new File(userContent + "lcdmarquees/" + system + "-" + named + ".jpg").exists()) {//ok no USB game marquee so let's look at internal now
            System.out.println("[INTERNAL USER CONTENT] Looking for User Content @ " + String.format("%slcdmarquees/%s-%s.jpg", userContent, system, named));
            LogMe.aLogger.info("[INTERNAL USER CONTENT] Looking for User Content @ " + String.format("%slcdmarquees/%s-%s.jpg", userContent, system, named));
            imageMarqueeFound = true;
            System.out.println("[INTERNAL USER CONTENT] FOUND " + String.format("%s-%s", system, named));
            LogMe.aLogger.info("[INTERNAL USER CONTENT] FOUND " + String.format("lcdmarquees/%s-%s.jpg", system, named));
            System.out.println("[INTERNAL USER CONTENT] switching {named} to " + String.format("%s-%s", system, named));
            named = String.format("%s-%s", system, named);
            System.out.println("[INTERNAL USER CONTENT] named is now " + named);
            OVERRIDE = wrapperHome + "gsho  -platform linuxfb \"" + userContent + "lcdmarquees/" + named + ".jpg\"";
            customContent = userContent;
            //marqueePath = userContent + "lcdmarquees/" + system + "-" + named + ".jpg";
            marqueePath = userContent + "lcdmarquees/" + named + ".jpg";
        } 
        else if ((ImagePathCaseSensitive = WebEnabledPixel.imageMarqueefileCache.get((pixelHome + "lcdmarquees/" + system + "-" + named + ".jpg").toLowerCase())) != null) {
        
            //ImagePathLowerCase =  (pixelHome + "lcdmarquees/" + system + "-" + named + ".jpg").toLowerCase();
            //ImagePathCaseSensitive = WebEnabledPixel.imageMarqueefileCache.get(ImagePathLowerCase);
            //if (ImagePathCaseSensitive != null) {  //we have a match so let's just get the path to the original case in case it's a case senstivie thing
                System.out.println("Original Case Sensitive File Name: " + ImagePathCaseSensitive);
                imageMarqueeFound = true;
                customContent = pixelHome;
                  //note we have to convert named which was the original rom name passed, we then did a case senstiive match, so now need to get back it's original case
                named = FileSystems.getDefault().getPath(ImagePathCaseSensitive).getFileName().toString().replaceFirst("[.][^.]+$", "");
                System.out.println("[INTERNAL SYSTEM] Found " + String.format("%slcdmarquees/%s.jpg",pixelHome, named));
                LogMe.aLogger.info("[INTERNAL SYSTEM] Found " + String.format("%slcdmarquees/%s.jpg",pixelHome, named));
                OVERRIDE = wrapperHome + "gsho  -platform linuxfb \"" + ImagePathCaseSensitive + "\"";
                marqueePath = "\"" + ImagePathCaseSensitive + "\""; //to do check that this is correct?
        } 
        else if (new File(String.format("%slcdmarquees/%s.jpg", pixelHome, named)).exists()) {  //looks for pacman.jpg as an example
                imageMarqueeFound = true;
                customContent = pixelHome;
                OVERRIDE = wrapperHome + "gsho  -platform linuxfb \"" + pixelHome + "lcdmarquees/" + named + ".jpg\"";
                marqueePath = String.format("%slcdmarquees/%s.jpg", pixelHome, named);
                // System.out.print(String.format("[INTERNAL] FOUND: %s.jpg in %slcdmarquees\n[mp: %s]\n[nf: %s]\n", named, pixelHome, , NOT_FOUND));
                 LogMe.aLogger.info(String.format("[INTERNAL] FOUND: %s.jpg in system storage", named));
        }
        else {
            System.out.println("[INFO] No JPG marquee match");
        }  
        
        
      
     //if (marqueePath.contains("default-")) {  //this means we only found the console and did not find the game marquee so let's try the fuzzy match
    //if (ConsoleFound == true && imageMarqueeFound == false) {  //so we only found the console and did not find the game marquee so let's try the fuzzy match    
    if (!imageMarqueeFound) {  //we did not find the game marquee so let's try the fuzzy match  
      
        if (!knownSets.containsKey(system)){
            ArrayList<String> workSet = new ArrayList<String>();
            for (String name : cleanList
            ) {
                if (name.startsWith(system + "-")) {
                    workSet.add(name);
                }
            }
            knownSets.put(system, workSet);
            System.out.print(String.format("[INTERNAL] ADDED WORKSET: %s with %d enties\n", system, workSet.size()));
        }

        String matchName = system + "-" + oldNamed;

        if (oldNamed.length() > 2) {

            System.out.println("[INTERNAL] FUZZY Looking For " + matchName + " in " + system + " workset");
            bestGuess = "";
            highestConfidence = 0;
            for (String filename : knownSets.get(system)
            ) {
                int matchVal = FuzzySearch.tokenSetRatio(filename, matchName);
                if (matchVal >= 90) {
                    System.out.println("[INTERNAL] I like: " + filename + " for this with a confidence of " + matchVal);
                    if (matchVal > 90 && matchVal > highestConfidence) {
                        bestGuess = filename;
                        highestConfidence = matchVal;
                    }
                }
            }

            if (!bestGuess.equals("")) {
                imageMarqueeFound = true;
                named = bestGuess.replace(".jpg", "");
                System.out.println("[INTERNAL] switching {named} to FUZZY " + bestGuess);
                LogMe.aLogger.info("[INTERNAL] found FUZZY match " + bestGuess);
                OVERRIDE = wrapperHome + "gsho  -platform linuxfb \"" + pixelHome + "lcdmarquees/" + named + ".jpg\"";
                marqueePath = String.format("%slcdmarquees/%s.jpg", pixelHome, named);
                oldNamed = named;
            }
        }
    }
      
   //System.out.println("[DEBUG] Marquee Found= " + imageMarqueeFound + " and Console Found = " + ConsoleFound); 

   theCommand = OVERRIDE;
   
    if (!WebEnabledPixel.getImageMarqueesOnly().contains("yes") && videoOKParam) { //changing to get this dynamically so we don't need to reboot
    //if (!imageMarqueesOnly_.contains("yes") && videoOKParam) { //skip this if video marquees are turned off or a param was sent to not show video
       
        if (new File(usbContent + "mp4marquees/" + oldNamed + ".mp4").exists()) {  //oldNames is the original param passed, it hasn't been changed //to do add 
            playVid = true;
            System.out.println("[USB VIDEO] Found " + String.format("%smp4marquees/%s.mp4",usbContent, oldNamed));
            LogMe.aLogger.info("[USB] Found " + String.format("%smp4marquees/%s.mp4",usbContent, oldNamed));
            videoLocation = userContent;
        } 
        else if (new File(userContent + "mp4marquees/" + oldNamed + ".mp4").exists()) {
            playVid = true;
            System.out.println("[INTERNAL USER VIDEO] Found " + String.format("%smp4marquees/%s.mp4",userContent, oldNamed));
            LogMe.aLogger.info("[INTERNAL USER VIDEO] Found " + String.format("%smp4marquees/%s.mp4",userContent, oldNamed));
            videoLocation = userContent;
        }
        else if (new File(pixelHome + "mp4marquees/" + oldNamed + "-mout.mp4").exists()) {
           playVid = true;
           MAMEOutputVideo = true;
           System.out.println("[INTERNAL MAME OUTPUT VIDEO] Found " + String.format("%smp4marquees/" + oldNamed + "-mout.mp4",pixelHome));
           LogMe.aLogger.info("[INTERNAL MAME OUTPUT VIDEO] Found " + String.format("%smp4marquees/" + oldNamed + "-mout.mp4",pixelHome));
           videoLocation = pixelHome;
        }
        else {
            VideoPathLowerCase =  (pixelHome + "mp4marquees/" + oldNamed + ".mp4").toLowerCase();
            VideoPathCaseSensitive = WebEnabledPixel.videoMarqueefileCache.get(VideoPathLowerCase);
            if (VideoPathCaseSensitive != null) {  //we have a match so let's just get the path to the original case in case it's a case senstivie thing
                System.out.println("Original Case Sensitive File Name: " + VideoPathCaseSensitive);
                playVid = true;
                //we have to convert oldNamed which was the original rom name passed to it's original case
                oldNamed = FileSystems.getDefault().getPath(VideoPathCaseSensitive).getFileName().toString().replaceFirst("[.][^.]+$", "");
                System.out.println("[INTERNAL USER VIDEO] Found " + String.format("%smp4marquees/%s.mp4",userContent, oldNamed));
                LogMe.aLogger.info("[INTERNAL USER VIDEO] Found " + String.format("%smp4marquees/%s.mp4",userContent, oldNamed));
                //System.out.println("[INTERNAL SYSTEM VIDEO] Found " + VideoPathCaseSensitive);
                //LogMe.aLogger.info("[INTERNAL SYSTEM VIDEO] Found " + VideoPathCaseSensitive);
                videoLocation = pixelHome;
            } else {
                System.out.println("\"[INTERNAL SYSTEM VIDEO] NOT Found");
                playVid = false;
                MAMEOutputVideo = false;
            }     
            
        }  
   
        if (playVid) {
            // MP4_COMMAND = "sudo mplayer -vo fbdev:/dev/fb0 \"${location}mp4marquees/${named}.mp4\" ; " + wrapperHome + "gsho -platform linuxfb \"${marqueeHome}lcdmarquees/${longNamed}.jpg\" ";
            if (imageMarqueeFound) {
                T = MP4_COMMAND.replace("${named}",oldNamed).replace("${longNamed}",named).replace("${location}", videoLocation).replace("${marqueeHome}", customContent);
            }
            else if (ConsoleFound) { //there is no marquee so let's then check for console
                T = MP4_COMMAND.replace("${named}",oldNamed).replace("${longNamed}","default-" + system).replace("${location}", videoLocation).replace("${marqueeHome}", customContent);
            }
            else if (MAMEOutputVideo) {
                T = MP4LOOP_COMMAND.replace("${named}",oldNamed + "-mout.mp4").replace("${location}", videoLocation);
            }
            else {  
                T = MP4_COMMAND.replace("${named}",oldNamed).replace("${longNamed}","pixelcade").replace("${location}", videoLocation).replace("${marqueeHome}", pixelHome);
            }
            
            theCommand = T;
            System.out.println("[[VIDEO CONTENT]] Switching to video playback, using: " + theCommand);
            LogMe.aLogger.info("[[VIDEO CONTENT]] Video playback, using: " + theCommand);
            runCommand(named, false, 0);
            return;
        }
   }  
        
        //if(marqueePath.equals(NOT_FOUND)) { //well we stil didn't find anything so let's take the default marquee
        if(!ConsoleFound && !imageMarqueeFound) { //well we stil didn't find anything so let's take the default marquee
            System.out.print(String.format("[MISSING] Could not find %s.jpg",named));
            LogMe.aLogger.info(String.format("[MISSING] Could not find %s.jpg",named));
            if (new File(pixelHome + "lcdmarquees/pixelcade.jpg").exists()){
                named = "vanity6";
                theCommand = wrapperHome + "gsho  -platform linuxfb " + pixelHome + "lcdmarquees/pixelcade.jpg";
                marqueePath = pixelHome + "lcdmarquees/pixelcade.jpg";
            } else {
                named = "pixelcade";
                doGif = false;
                theCommand = DEFAULT_COMMAND;
                System.out.println("[INTERNAL] Switching to video playback for the default video since nothing else was found");
            }
            System.out.println("[INTERNAL] Changed 'named' to: " + named +"\n");
        }
        
     
        if (marqueeOverlay && !marqueePath.isEmpty()) {  //if marquee overlays is turned on AND marquee path is not empty
                theCommand = pixelHome + "dtext -text=\"" + overlayText + "\" -background=" + marqueePath + " -font=\"" + pixelHome
                + "fonts/" + fontPath + "\"" + " -overlay"; //overlay is the flag to tell us to overlay on the game marquee
                
//                  theCommand = pixelHome + "dtext -text=\"" + overlayText + "\" -background=" + marqueePath + " -font=\"" + pixelHome
//                + "fonts/" + "Nintendo DS BIOS.ttf" + "\"" + " -overlay"; //overlay is the flag to tell us to overlay on the game marquee
        }
       
        runCommand(named, false, 0);
        
        
    }

    
    
//    static public void preGameScrollText(String message, Font font, Color color, int speed, String system, String named) {
//            delayedImage(system, named);  //we want this shown after the delay
//            scrollText(message,font,color,speed); //now automatically has delay
//    }
    
    static public void playAchievement(String message, Font font, Color color, int speed, String system, String named) {

        delayedImage(system, named);  //we want this shown after the delay
        try {
            AchievementscrollText(message,font,color,speed, system, named); //now automatically has delay
        } catch (IOException ex) {
            Logger.getLogger(LCDPixelcade.class.getName()).log(Level.SEVERE, null, ex);
        }
    }
    
    static public void playGameLaunchscrollText(String message, Font font, Color color, int speed, String system, String named) {

        delayedImage(system, named);  //we want this shown after the delay
        try {
            GameLaunchscrollText(message,font,color,speed, system, named); //now automatically has delay
        } catch (IOException ex) {
            Logger.getLogger(LCDPixelcade.class.getName()).log(Level.SEVERE, null, ex);
        }
    }
     
    static public void GameLaunchscrollText(String message, Font font, Color color, int speed, String system, String named) throws IOException {

        WebEnabledPixel.setLCDFont();
          
         //so we don't need to reboot after setting font
        WebEnabledPixel.setNowPlayingFlag(true); //set the atomic to true so we know there is a splash screen happen and we'll use it to ignore the next call
        // Count files matching the pattern background-xx.jpg
        int fileCount = countMatchingFiles(pixelHome + "backgrounds/", "background-\\d{2}\\.jpg");
        //System.out.println("Number of matching files: " + fileCount);

        if (fileCount > 0) {
            
            String GameLaunchBackground = getRandomMatchingFile(pixelHome + "backgrounds/", "background-\\d{2}\\.jpg");
            Path GameLaunchBackgroundPath = Paths.get(GameLaunchBackground);
            GameLaunchBackgroundFileName = GameLaunchBackgroundPath.getFileName().toString();
            
            String hex = String.format("#%02x%02x%02x", color.getRed(), color.getGreen(), color.getBlue());
            //message = message + "  ";
            
            //first let's check if we have high scores, also let's blank out ~highscore
            
            String NowPlaying_ = "";
            HighScores_ = "";
            
            //let's also remove  ~hiscoretop3~ ~hiscore~ ~hiscoreall~, this is kind of a hack, it's a fail safe in case LEDBlinky substituion did not work which I've seen happen
            message = message
                .replace("~hiscoretop3~", "")
                .replace("~hiscore~", "")
                .replace("~hiscoreall~", "");
            
            if (message.contains("#1")) { //this means we have a high score
                // Split the string using the first occurrence of "#1"
                String[] splitParts = message.split("#1", 2);

                // Check if the split resulted in two parts
                if (splitParts.length == 2) {
                     NowPlaying_ = splitParts[0];
                     HighScores_ = "#1" + splitParts[1];
                     lcdMessage = NowPlaying_ + "\n" + "HIGH SCORES\n" + HighScores_ + "\n";
                     dTextShowTime = HighScoreAchievementsPlayingShowTime; //becasue there are high scores, let's give the user some more time to read them
                     DotMessage = HighScores_;
                     highScoreDotFlag = true; // had to add this so when the normal marquee plays , we still show high scores on Pixelcade Dot
                     
//                     theCommand = pixelHome + "dtext -text=" + "\"" + lcdMessage + "\"" + " -background=" + pixelHome
//                        + "backgrounds/" + GameLaunchBackgroundFileName +  " -font=" + pixelHome 
//                        + "fonts/" + "Eight-Bit-Madness.ttf" + " -top-margin=60.0";
                     
                       theCommand = pixelHome + "dtext -text=\"" + lcdMessage + "\" -background=" + pixelHome
                            + "backgrounds/" + GameLaunchBackgroundFileName + " -top-margin=60.0 -font=\"" + pixelHome
                            + "fonts/" + fontPath + "\"";
                     
                } else {
                    System.out.println("The input string does not contain '#1'");
                    lcdMessage = message;
                    DotMessage = message;
                    dTextShowTime = NowPlayingShowTime; 
                    //theCommand = pixelHome + "dtext -text=" + "\"" + lcdMessage + "\"" + " -background=" + pixelHome + "backgrounds/" + GameLaunchBackgroundFileName + " -top-margin=60.0";
                    
                    theCommand = pixelHome + "dtext -text=\"" + lcdMessage + "\" -background=" + pixelHome
                        + "backgrounds/" + GameLaunchBackgroundFileName + " -top-margin=60.0 -font=\"" + pixelHome
                        + "fonts/" + fontPath + "\"";
                }
            } 
            else {
                lcdMessage = message;
                DotMessage = message;
                dTextShowTime = NowPlayingShowTime; 
                //theCommand = pixelHome + "dtext -text=" + "\"" + lcdMessage + "\"" + " -background=" + pixelHome + "backgrounds/" + GameLaunchBackgroundFileName + " -top-margin=60.0";
                theCommand = pixelHome + "dtext -text=\"" + lcdMessage + "\" -background=" + pixelHome
                    + "backgrounds/" + GameLaunchBackgroundFileName + " -top-margin=60.0 -font=\"" + pixelHome
                    + "fonts/" + fontPath + "\"";
            }
            
            fontColor = hex;
            
            if (WebEnabledPixel.getDotMatrixExists()) {
                if (DotMessage.length() < 8) {
                    DotMessage = String.format("%-8s", DotMessage);
                }
                pCadeMicro.scrollText(DotMessage,10000); 
            }
            
            runCommand("dtext-output", false, dTextShowTime); //let's generate the scrolling text jpg and show it for 5 seconds
            //we pause execution here for 10 seconds!
          
        } else {
            System.out.println("[ERROR] No matching background files found.");
            LogMe.aLogger.info("[ERROR] No matching background files found, please run an artwork update");
        }
        
        //String myCommand = TXT_COMMAND.replace("${txt}",message).replace("${fontpath}",fontPath.replace(".ttf","")).replace("${color}",fontColor).replace("${speed}",String.format("%d",loops));
        String myNamed = String.format("%s-%s",delayedSystem,delayedName);
        
        isScrolling = false;
        RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho; killall -9 skrola;";
        
        if (WebEnabledPixel.getMarqueeOverlays().equals("yes") && !HighScores_.isEmpty()) { //we dont' want to do overlay for now playing splash screen, only high scores
            displayImage(delayedName,delayedSystem,false, true,"High Scores: " + HighScores_); //now display the original marquee //to do saw a few times that params were not passed and landed on generic marquee
        } else {
            displayImage(delayedName,delayedSystem,true, false,""); 
        }
        
        delayedName = "";
        delayedSystem = "";
        isScrolling = false;
       
    }
    
        static public void ShowStats(String levelLabel, int levelValue, int shotsValue, int hitsValue, int ratioValue) throws IOException {
           dTextShowTime = NowPlayingShowTime;

           // Start with required parameters
           StringBuilder commandBuilder = new StringBuilder();
           commandBuilder.append(pixelHome)
                        .append("dtext -stats")
                        .append(" -stage-label=\"").append(levelLabel).append("\"")
                        .append(" -stage=").append(levelValue);

           // Add optional parameters if they were provided (not -1)
           if (shotsValue != -1) {
               commandBuilder.append(" -shots=").append(shotsValue);
           }
           if (hitsValue != -1) {
               commandBuilder.append(" -hits=").append(hitsValue);
           }
           if (ratioValue != -1) {
               commandBuilder.append(" -ratio=").append(ratioValue);
           }

           theCommand = commandBuilder.toString();

           String myNamed = String.format("%s-%s", delayedSystem, delayedName);

           isScrolling = false;
           RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho; killall -9 skrola; killall -9 dtext;";

           runCommand("dtext-output", false, dTextShowTime);
       }
      
      static public void runCommand(String named, boolean waitToComplete, int postWaitTime) throws IOException {  //note this is Pi/linux only!
        
        if (named == null) return;

        System.out.println("[INTERNAL] 'named' for displayImage is: " + named +"\n");
        System.out.println("[INTERNAL] 'theCommand' is: " + theCommand +"\n");

        if(playVid) {
            System.out.println("[INTERNAL] Prepping for Video");
        }
        
       if (slideshowPlayer.isPlaying() || isScrolling){  //let's skip the marquee call if the slideshow is playing OR if we're scrolling text ?
            slideshowPlayer.stop();
            System.out.println("Ending Attract mode");
        }

        ProcessBuilder builder = new ProcessBuilder();
        builder.command("sh", "-c", RESET_COMMAND + theCommand);
        System.out.println("[RUNNING COMMAND] " + "sh -c " +  RESET_COMMAND + theCommand);
        Process process = builder.start();
        
        if (waitToComplete) {  //this is just for achievements video to let the video finish but it's a flag passed so not always will delay
            System.out.println("We're waiting for the video to complete before we continue");
            while (process.isAlive()) {
             }
        }
        
        if (postWaitTime > 0) { //for dtext 
        
         try {
            //if it's a now playing / high scores splash screen before the game marquee
            
            
            Thread.sleep(postWaitTime*1000);
        
//           nowPlayingShowing.set(true);
//           future = CompletableFuture.runAsync(() -> {
//            while (nowPlayingShowing.get()) {
//                // Your asynchronous loop logic
//                System.out.println("Async loop running Now Showing Flag = " + nowPlayingShowing.get());
//                try {
//                    TimeUnit.SECONDS.sleep(1);
//                } catch (InterruptedException e) {
//                    Thread.currentThread().interrupt();
//                    break; // Exit the loop on interruption
//                }
//            }
//        }, CompletableFuture.delayedExecutor(postWaitTime, TimeUnit.SECONDS));
//
//        nowPlayingShowing.set(false);
//
//        // Wait for the delay or interrupt before launching the next process
//        //future.thenRun(() -> launchNextProcess()); //next part was suggested if the nextprocess was getting cancelled but so far it seems to be ok without it
//        
//           future.thenRun(() -> launchNextProcess())
//              .exceptionally(throwable -> {
//                  if (throwable instanceof CancellationException) {
//                      System.out.println("Action canceled due to interruption.");
//                  } else {
//                      throwable.printStackTrace();
//                  }
//                  return null;
//              });
//            
//                    nowPlayingShowing.set(true);
//                    future = CompletableFuture.runAsync(() -> {
//                    while (nowPlayingShowing.get()) {
//                        // Your asynchronous loop logic
//                        System.out.println("Async loop running Now Showing Flag = " + nowPlayingShowing.get());
//                        try {
//                            TimeUnit.SECONDS.sleep(1);
//                        } catch (InterruptedException e) {
//                            Thread.currentThread().interrupt();
//                        }
//                    }
//                });
            } catch (InterruptedException ex) {
                Logger.getLogger(LCDPixelcade.class.getName()).log(Level.SEVERE, null, ex);
            }
            
        } 

        if(playVid) {
            playVid = false;
            MAMEOutputVideo = false;
        } //reset it for the next time
        

    }
      
      /* 
      public static void launchNextProcess() {  //this didn't work so not using it
        //System.out.println("Launching the next process after the delay or interrupt.");
        // System.out.println("Now go back to game marquee...");
            isScrolling = false;
            RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho; killall -9 skrola;";
        try {
            displayImage(delayedName,delayedSystem,false,false); //now display the original marquee //to do saw a few times that params were not passed and landed on generic marquee
        } catch (IOException ex) {
            Logger.getLogger(LCDPixelcade.class.getName()).log(Level.SEVERE, null, ex);
        }
            delayedName = "";
            delayedSystem = "";
            isScrolling = false;
    }*/
      
   
      
//       public static void stopAsyncLoop() {  //also didn't work
//           
//        // Optionally, cancel the CompletableFuture to interrupt the asynchronous loop
//        if (future != null) {
//             // Complete the CompletableFuture to allow the get method to return
//            future.complete(null);
//            future.cancel(true);
//            
//            System.out.println("[CANCEL ASYNC] LOOP");
//            nowPlayingShowing.set(false);
//        }
//        else {
//             System.out.println("Future was null so doing nothing");
//        }
//       
//    }
      

     static public void AchievementscrollText(String message, Font font, Color color, int speed, String system, String named) throws IOException {

        WebEnabledPixel.setLCDFont();

        // Count files matching the pattern achievements-xx.mp4
        int fileCount = countMatchingFiles(pixelHome + "mp4marquees/", "achievements-\\d{2}\\.mp4");
        //System.out.println("Number of matching files: " + fileCount);

        if (fileCount > 0) {
            // Randomly pick one of the matching files
            String achievementsVideo = getRandomMatchingFile(pixelHome + "mp4marquees/", "achievements-\\d{2}\\.mp4");
            // example: /home/pi/pixelcade/mp4marquees/achievements-19.mp4
            //System.out.println("Randomly selected Achievements Video: " + achievementsVideo);
            //ok we found an achivements video file so let's play it
             // Create a Path object from the file path
             Path achievementsVideoPath = Paths.get(achievementsVideo);
            // Get the file name without extension
            String achievementBaseNameVideo = achievementsVideoPath.getFileName().toString().replaceFirst("[.][^.]+$", "");
            
            String achievementsJPG = getRandomMatchingFile(pixelHome + "backgrounds/", "achievements-\\d{2}\\.jpg");
            Path achievementsJPGPath = Paths.get(achievementsJPG);
            AchievementBackgroundFileName = achievementsJPGPath.getFileName().toString();
            //System.out.println("[PATH] Randomly selected Achievements JPG: " + AchievementBackgroundFileName); //achievements-02.jpg
            theCommand = MP4ONLY_COMMAND.replace("${named}",achievementBaseNameVideo).replace("${location}", pixelHome);
            
            String hex = String.format("#%02x%02x%02x", color.getRed(), color.getGreen(), color.getBlue());
            message = message + "  ";
            
            String DotAchievementMessage = "Achievement Unlocked: " + message; //we dont' want the newline character for pixelcade dot
            fontColor = hex;
            
            if (WebEnabledPixel.getDotMatrixExists()) {
                if (DotAchievementMessage.length() < 8) {
                    DotAchievementMessage = String.format("%-8s", DotAchievementMessage);
                }
                pCadeMicro.scrollText(DotAchievementMessage,10000);
            }
            
            System.out.println("[[VIDEO CONTENT achievement]] Switching to video playback, using: " + theCommand);
            LogMe.aLogger.info("[[VIDEO CONTENT achievemebt]] Video playback, using: " + theCommand);
           
            runCommand("achievements-video", true, 0); //this is the achievements video, let's wait for it to complete before we show the next thing
          
        } else {
            System.out.println("No matching files found.");
        }

        //message = "Achievement Unlocked\n" + message; //dont' need this now as the backbround already has the Achievement Unlocked text
        //theCommand = pixelHome + "dtext -text=" + "\"" + message + "\"" + " -background=" + pixelHome + "backgrounds/" + AchievementBackgroundFileName + " -top-margin=160.0";
        
        
        theCommand = pixelHome + "dtext -text=\"" + message + "\" -background=" + pixelHome
        + "backgrounds/" + AchievementBackgroundFileName + " -top-margin=160.0 -font=\"" + pixelHome
        + "fonts/" + fontPath + "\"";
        
       
        runCommand("dtext-output", false, HighScoreAchievementsPlayingShowTime); //let's generate the achievements text and wait 10 seconds
        
        //String myCommand = TXT_COMMAND.replace("${txt}",message).replace("${fontpath}",fontPath.replace(".ttf","")).replace("${color}",fontColor).replace("${speed}",String.format("%d",loops));
        String myNamed = String.format("%s-%s",delayedSystem,delayedName);
        
        //System.out.println("Now go back to game marquee...");
        isScrolling = false;
        RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho; killall -9 skrola;";
        
        if (WebEnabledPixel.getMarqueeOverlays().equals("yes")) {
            displayImage(delayedName,delayedSystem,false,true,message); 
        } else {
            displayImage(delayedName,delayedSystem,false,false,""); 
        }
        
        delayedName = "";
        delayedSystem = "";
        isScrolling = false;
    }
     
     static public void scrollText(String message, Font font, Color color, int speed) {

        WebEnabledPixel.setLCDFont(); //we need this so we can get the latest font without rebooting
        
        int BackgroundFileCount = countMatchingFiles(pixelHome + "backgrounds/", "background-\\d{2}\\.jpg");
        //System.out.println("Number of matching files: " + fileCount);

        if (BackgroundFileCount > 0) {
            String TextBackground = getRandomMatchingFile(pixelHome + "backgrounds/", "background-\\d{2}\\.jpg");
            Path TextBackgroundPath = Paths.get(TextBackground);
            TextBackgroundFileName = TextBackgroundPath.getFileName().toString();
        } 
        else {
            TextBackgroundFileName = pixelHome + "backgrounds/background.jpg";
        }
         
         
        String hex = String.format("#%02x%02x%02x", color.getRed(), color.getGreen(), color.getBlue());
        message = message + "  ";
        fontColor = hex;
        //String myCommand = TXT_COMMAND.replace("${txt}",message).replace("${fontpath}",fontPath.replace(".ttf","")).replace("${color}",fontColor).replace("${speed}",String.format("%d",loops));
        //String myNamed = String.format("%s-%s",delayedSystem,delayedName);
        
        playVid = false;

        if (WebEnabledPixel.getDotMatrixExists()) {
            String finalMessage = message;
            if (finalMessage.length() < 8) {
                    finalMessage = String.format("%-8s", finalMessage);
            }
            pCadeMicro.scrollText(finalMessage,10000);
        }
        //add font path and font color
        
        //System.out.println("[[Font]]  " + fontPath + "color " + fontColor + pixelHome + "fonts/" + fontPath);
        
        
       
        //theCommand = pixelHome + "dtext -text=" + "\"" + message + "\"" + " -background=" + pixelHome + "backgrounds/" + TextBackgroundFileName + " -top-margin=60.0" + " -font=" + pixelHome + "fonts/" + fontPath;
        theCommand = pixelHome + "dtext -text=\"" + message + "\" -background=" + pixelHome
        + "backgrounds/" + TextBackgroundFileName + " -top-margin=60.0 -font=\"" + pixelHome
        + "fonts/" + fontPath + "\"";
        
        try {
            runCommand("dtext-output", false, NowPlayingShowTime); //let's generate the achievements text and wait 10 seconds
        } catch (IOException ex) {
            Logger.getLogger(LCDPixelcade.class.getName()).log(Level.SEVERE, null, ex);
        }
                
    }
     
//     static public void scrollText(String message, Font font, Color color, int speed) {  //old scroll text code with skrola
//
//        String hex = String.format("#%02x%02x%02x", color.getRed(), color.getGreen(), color.getBlue());
//        message = message + "  ";
//        fontColor = hex;
//        String myCommand = TXT_COMMAND.replace("${txt}",message).replace("${fontpath}",fontPath.replace(".ttf","")).replace("${color}",fontColor).replace("${speed}",String.format("%d",loops));
//        String myNamed = String.format("%s-%s",delayedSystem,delayedName);
//  
//        //LCDPixelcade.theCommand = JPG_COMMAND.replace("${named}",myNamed); // this was a bug had to fix as there is no check to ensure the file is there
//        
//        //theCommand = "killall -9 gsho; " + JPG_COMMAND.replace("${named}",named) + "&; " + theCommand;
//            if(loops > 0 && loops < ENDOFQ) {
//
//                String finalTheCommand = myCommand;
//                playVid = false;
//
//                String finalMessage = message;
//                //pCadeMicro.scrollText(finalMessage,loops);
//                pCadeMicro.scrollText(finalMessage,10000);
//                Thread thread = new Thread(() -> {
//                    try {
//                        RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho;";
//                        //RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho; killall -9 skrola;"; //this results in no scrolling text
//
//                        
//                        displayImage(delayedName,delayedSystem,false); //this was the bug fix, call display image to check for path but with a flag to only play a JPG and no video
//                        //runCommand(myNamed);
//
//                        ProcessBuilder builder = new ProcessBuilder();
//                        builder.command("sh", "-c", finalTheCommand);
//                        System.out.println("Generating & scrolling");
//                        System.out.println(finalTheCommand);
//                        LogMe.aLogger.info("Generating & scrolling");
//                        LogMe.aLogger.info(finalTheCommand);
//                        if(!delayedSystem.equals("")){
//                            runCommand(myNamed,false,0);
//                        }
//
//                        isScrolling = true;
//                        process = builder.start();
//                        while (process.isAlive()) {
//                        }
//
//                        System.out.println("Displaying...");
//                        isScrolling = false;
//                        RESET_COMMAND = "sudo killall -9 mplayer;killall -9 gsho; killall -9 skrola;";
//                        displayImage(delayedName,delayedSystem,true);
//                        delayedName = "";
//                        delayedSystem = "";
//                        } catch (IOException e) {
//                            isScrolling = false;
//                            e.printStackTrace();
//                        }
//                    });
//                    thread.start();
//            }
//                isScrolling = false;
//    }
     
  
   
    
    private static int countMatchingFiles(String directoryPath, String pattern) {
        File directory = new File(directoryPath);
        File[] files = directory.listFiles((dir, name) -> name.matches(pattern));

        return files != null ? files.length : 0;
    }

    private static String getRandomMatchingFile(String directoryPath, String pattern) {
        File directory = new File(directoryPath);
        File[] files = directory.listFiles((dir, name) -> name.matches(pattern));
        long seed = System.currentTimeMillis();
        if (files != null && files.length > 0) {
            Random random = new Random(seed);
            int randomIndex = random.nextInt(files.length);
            return files[randomIndex].getAbsolutePath();
        }

        return null;
    }
}
