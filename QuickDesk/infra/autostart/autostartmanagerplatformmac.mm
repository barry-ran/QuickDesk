#include "autostartmanagerplatform.h"

#import <Foundation/Foundation.h>
#import <CoreServices/CoreServices.h>

#include <QString>

namespace infra {

namespace {

bool checkLoginItems(bool removeItem)
{
    BOOL result = NO;
    
    NSString* appPath = [[NSBundle mainBundle] bundlePath];
    CFURLRef urlRef = nil;
    
    LSSharedFileListRef loginItems = LSSharedFileListCreate(nil, kLSSharedFileListSessionLoginItems, nil);
    if (!loginItems) {
        return false;
    }
    
    UInt32 seed;
    CFArrayRef loginItemsArrayRef = LSSharedFileListCopySnapshot(loginItems, &seed);
    if (!loginItemsArrayRef) {
        CFRelease(loginItems);
        return false;
    }
    
    for (id item in (__bridge NSArray*)loginItemsArrayRef) {
        LSSharedFileListItemRef itemRef = (__bridge LSSharedFileListItemRef)item;
        if (noErr != LSSharedFileListItemResolve(itemRef, 0, (CFURLRef*)&urlRef, nil)) {
            continue;
        }
        if ([[(__bridge NSURL*)urlRef path] hasPrefix:appPath]) {
            result = YES;
            if (removeItem) {
                LSSharedFileListItemRemove(loginItems, itemRef);
            }
        }
        if (urlRef) {
            CFRelease(urlRef);
            urlRef = nil;
        }
        if (result) {
            break;
        }
    }
    
    if (loginItemsArrayRef) {
        CFRelease(loginItemsArrayRef);
    }
    if (loginItems) {
        CFRelease(loginItems);
    }
    
    return result == YES;
}

bool addToLoginItems()
{
    NSString* appPath = [[NSBundle mainBundle] bundlePath];
    CFURLRef urlRef = (__bridge CFURLRef)[NSURL fileURLWithPath:appPath];
    
    CFMutableDictionaryRef inPropertiesToSet = CFDictionaryCreateMutable(nil, 1, nil, nil);
    CFDictionaryAddValue(inPropertiesToSet, kLSSharedFileListLoginItemHidden, kCFBooleanTrue);
    
    LSSharedFileListRef loginItems = LSSharedFileListCreate(nil, kLSSharedFileListSessionLoginItems, nil);
    if (!loginItems) {
        CFRelease(inPropertiesToSet);
        return false;
    }
    
    LSSharedFileListItemRef itemRef = LSSharedFileListInsertItemURL(
        loginItems, kLSSharedFileListItemLast, nil, nil, urlRef, inPropertiesToSet, nil);
    
    CFRelease(inPropertiesToSet);
    
    bool success = (itemRef != nil);
    
    if (itemRef) {
        CFRelease(itemRef);
    }
    if (loginItems) {
        CFRelease(loginItems);
    }
    
    return success;
}

} // namespace

bool AutoStartManagerPlatform::isAutoStartEnabled()
{
    return checkLoginItems(false);
}

bool AutoStartManagerPlatform::enableAutoStart(const QString& /* args */)
{
    checkLoginItems(true);
    return addToLoginItems();
}

bool AutoStartManagerPlatform::disableAutoStart()
{
    return checkLoginItems(true);
}

}
