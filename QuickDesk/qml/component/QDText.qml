// Fluent Design Text Component
// Themed text display with typography presets from the Fluent type ramp.
import QtQuick

Text {
    id: control
    
    // ============ Typography Type Ramp ============
    
    enum Type {
        Caption,     // 12px — Small annotations, timestamps
        Body,        // 14px — Default body text (default)
        Subtitle,    // 16px — Section subtitles
        BodyLarge,   // 18px — Emphasized body text
        Title,       // 20px — Page/section titles
        Heading      // 24px — Major headings
    }
    
    // ============ Color Variants ============
    
    enum ColorRole {
        Primary,     // Theme.text         — Default, high emphasis
        Secondary,   // Theme.textSecondary — Medium emphasis
        Disabled,    // Theme.textDisabled  — Low emphasis / disabled
        OnPrimary,   // Theme.textOnPrimary — Text on colored backgrounds
        Custom       // Uses customColor property
    }
    
    // ============ Custom Properties ============
    
    property int type: QDText.Type.Body
    property int colorRole: QDText.ColorRole.Primary
    property color customColor: Theme.text
    property bool mono: false  // Use monospace font (Theme.fontFamilyMono)
    
    // ============ Style (auto-derived from type/colorRole) ============
    
    font.family: mono ? Theme.fontFamilyMono : Theme.fontFamily
    
    font.pixelSize: {
        switch (type) {
            case QDText.Type.Caption:   return Theme.fontSizeSmall
            case QDText.Type.Body:      return Theme.fontSizeMedium
            case QDText.Type.Subtitle:  return Theme.fontSizeLarge
            case QDText.Type.BodyLarge: return Theme.fontSizeXLarge
            case QDText.Type.Title:     return Theme.fontSizeTitle
            case QDText.Type.Heading:   return Theme.fontSizeHeading
            default:                    return Theme.fontSizeMedium
        }
    }
    
    font.weight: {
        switch (type) {
            case QDText.Type.Title:   return Font.DemiBold
            case QDText.Type.Heading: return Font.Bold
            default:                  return Font.Normal
        }
    }
    
    color: {
        switch (colorRole) {
            case QDText.ColorRole.Primary:   return Theme.text
            case QDText.ColorRole.Secondary: return Theme.textSecondary
            case QDText.ColorRole.Disabled:  return Theme.textDisabled
            case QDText.ColorRole.OnPrimary: return Theme.textOnPrimary
            case QDText.ColorRole.Custom:    return customColor
            default:                         return Theme.text
        }
    }
}
