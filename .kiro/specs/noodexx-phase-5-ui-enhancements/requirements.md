# Requirements Document: Noodexx Phase 5 UI Enhancements

## Introduction

This document specifies the requirements for modernizing the Noodexx user interface by integrating Tailwind CSS, Alpine.js, and the existing HTMX framework. The enhancement maintains the Go template architecture while introducing a centralized theme configuration system and per-user dark mode preferences. The system provides a modern, accessible, and maintainable UI without requiring a build pipeline.

## Glossary

- **UIStyle_Manager**: Component that loads and manages the centralized theme configuration from uistyle.json
- **Theme_Configuration**: The centralized design system defined in uistyle.json including colors, typography, spacing, shadows, and border radius
- **Page_Shell**: The base.html template that provides foundational layout structure with dependency loading and theme management
- **Component**: Reusable Go template partial for UI elements (button, card, modal, toast, etc.)
- **Dark_Mode**: User interface theme with dark background colors, stored per-user in the database
- **Color_Palette**: A set of color shades from 50 (lightest) to 900 (darkest) for a semantic color (primary, secondary, etc.)
- **CDN**: Content Delivery Network used to load Tailwind CSS and Alpine.js
- **Toast**: Non-intrusive notification message with auto-dismiss behavior
- **Modal**: Dialog overlay for confirmations, forms, and detail views
- **HTMX**: HTML-driven AJAX library for partial page updates
- **Alpine.js**: Lightweight JavaScript framework for reactive UI components
- **Tailwind_CSS**: Utility-first CSS framework for styling

## Requirements

### Requirement 1: Centralized Theme Configuration

**User Story:** As a system administrator, I want to define all visual styling in a single configuration file, so that I can maintain a consistent design system across the application.

#### Acceptance Criteria

1. THE UIStyle_Manager SHALL load theme configuration from uistyle.json at application startup
2. WHEN uistyle.json is missing or malformed, THEN THE UIStyle_Manager SHALL prevent application startup and return a descriptive error message
3. THE Theme_Configuration SHALL include color palettes for primary, secondary, success, warning, error, info, and surface colors
4. THE Theme_Configuration SHALL include typography configuration with font families and font sizes
5. THE Theme_Configuration SHALL include spacing configuration with unit and scale definitions
6. THE Theme_Configuration SHALL include border radius values for none, sm, base, md, lg, xl, and full
7. THE Theme_Configuration SHALL include shadow definitions for sm, base, md, lg, and xl
8. THE UIStyle_Manager SHALL inject the theme configuration into the Tailwind CSS config in all page templates
9. THE Theme_Configuration SHALL apply to all users consistently

### Requirement 2: Color Palette Validation

**User Story:** As a system administrator, I want color palettes to be validated at startup, so that invalid configurations are caught before they affect users.

#### Acceptance Criteria

1. THE UIStyle_Manager SHALL validate that all color palettes include shades 50, 100, 200, 300, 400, 500, 600, 700, 800, and 900
2. THE UIStyle_Manager SHALL validate that all color values are valid hexadecimal color codes in #RRGGBB format
3. WHEN a color palette is missing a required shade, THEN THE UIStyle_Manager SHALL return an error indicating the palette name and missing shade
4. WHEN a color value is not a valid hex code, THEN THE UIStyle_Manager SHALL return an error indicating the palette name, shade, and invalid value
5. THE UIStyle_Manager SHALL validate all seven color palettes (primary, secondary, success, warning, error, info, surface) before allowing application startup

### Requirement 3: Per-User Dark Mode Preference

**User Story:** As a user, I want my dark mode preference to persist across sessions and devices, so that I have a consistent experience.

#### Acceptance Criteria

1. THE System SHALL store each user's dark mode preference in the users table as a boolean column
2. WHEN a user loads any page, THEN THE System SHALL retrieve that user's dark mode preference from the database
3. WHEN a user toggles dark mode, THEN THE System SHALL immediately update the UI to reflect the new theme
4. WHEN a user toggles dark mode, THEN THE System SHALL persist the preference to the database via API call
5. IF the API call to persist dark mode fails, THEN THE System SHALL revert the UI to the previous theme state and display an error toast
6. THE System SHALL initialize the dark mode state from the database on page load, not from localStorage or OS preference
7. WHEN a user logs in from a different device or browser, THEN THE System SHALL apply that user's saved dark mode preference

### Requirement 4: Page Shell and Dependency Loading

**User Story:** As a developer, I want a consistent page shell with reliable dependency loading, so that all pages have the same foundation and work offline.

#### Acceptance Criteria

1. THE Page_Shell SHALL load Tailwind CSS from CDN with fallback to local static file
2. THE Page_Shell SHALL load Alpine.js from CDN with fallback to local static file
3. THE Page_Shell SHALL load HTMX from local static file
4. WHEN a CDN resource fails to load, THEN THE Page_Shell SHALL automatically load the vendored copy from web/static/
5. THE Page_Shell SHALL inject the UIStyle configuration into the Tailwind CSS config via inline script
6. THE Page_Shell SHALL initialize Alpine.js with the user's dark mode preference from the database
7. THE Page_Shell SHALL provide a global toast notification container
8. THE Page_Shell SHALL render a fixed sidebar and topbar on all authenticated pages

### Requirement 5: Button Component

**User Story:** As a developer, I want a reusable button component with consistent styling, so that all buttons in the application look and behave the same way.

#### Acceptance Criteria

1. THE Button_Component SHALL support variants: primary, secondary, danger, and ghost
2. THE Button_Component SHALL support sizes: sm, md, and lg
3. THE Button_Component SHALL support a loading state that displays a spinner and disables the button
4. THE Button_Component SHALL support a disabled state
5. THE Button_Component SHALL support button types: button and submit
6. WHEN an invalid variant is provided, THEN THE Button_Component SHALL default to primary variant
7. WHEN an invalid size is provided, THEN THE Button_Component SHALL default to md size
8. THE Button_Component SHALL accept additional CSS classes for customization
9. THE Button_Component SHALL render accessible HTML with proper button semantics

### Requirement 6: Card Component

**User Story:** As a developer, I want a reusable card component for content sections, so that content containers have consistent styling and dark mode support.

#### Acceptance Criteria

1. THE Card_Component SHALL render a container with background, border, shadow, and padding
2. THE Card_Component SHALL support an optional title that renders as a heading
3. THE Card_Component SHALL support dark mode with appropriate background and border colors
4. THE Card_Component SHALL accept additional CSS classes for customization
5. THE Card_Component SHALL render the provided content inside the card container

### Requirement 7: Modal Component

**User Story:** As a developer, I want a reusable modal component for dialogs, so that confirmations and forms have consistent behavior and accessibility.

#### Acceptance Criteria

1. THE Modal_Component SHALL render a dialog overlay with backdrop
2. THE Modal_Component SHALL trap focus within the modal when open
3. THE Modal_Component SHALL move focus to the first focusable element when opened
4. WHEN the user clicks the backdrop, THEN THE Modal_Component SHALL close the modal
5. WHEN the user presses the Escape key, THEN THE Modal_Component SHALL close the modal
6. THE Modal_Component SHALL include aria-modal="true" and role="dialog" attributes
7. THE Modal_Component SHALL support a title, content area, and action buttons
8. THE Modal_Component SHALL animate in and out with transitions
9. THE Modal_Component SHALL support dark mode styling

### Requirement 8: Toast Notification System

**User Story:** As a user, I want non-intrusive notifications for actions, so that I receive feedback without disrupting my workflow.

#### Acceptance Criteria

1. THE Toast_System SHALL display toast notifications with variants: success, error, info, and warning
2. THE Toast_System SHALL auto-dismiss toasts after a configurable duration (default 5000ms)
3. THE Toast_System SHALL enforce a maximum of 4 visible toasts at any time
4. WHEN a fifth toast is added, THEN THE Toast_System SHALL remove the oldest toast
5. THE Toast_System SHALL allow manual dismissal via a close button
6. THE Toast_System SHALL stack toasts vertically in the top-right corner
7. THE Toast_System SHALL animate toasts in and out with transitions
8. THE Toast_System SHALL support triggering toasts from backend via HTMX HX-Trigger header
9. THE Toast_System SHALL support dark mode styling

### Requirement 9: Empty State Component

**User Story:** As a developer, I want a reusable empty state component, so that empty lists and grids have consistent, helpful messaging.

#### Acceptance Criteria

1. THE Empty_State_Component SHALL display an icon, heading, and description
2. THE Empty_State_Component SHALL support an optional action button with text and URL
3. THE Empty_State_Component SHALL support dark mode styling
4. THE Empty_State_Component SHALL be visually centered in its container

### Requirement 10: Loading State Component

**User Story:** As a developer, I want a reusable loading state component, so that async content has consistent loading indicators.

#### Acceptance Criteria

1. THE Loading_State_Component SHALL support two types: skeleton and spinner
2. WHEN type is skeleton, THEN THE Loading_State_Component SHALL render animated placeholder rows
3. WHEN type is spinner, THEN THE Loading_State_Component SHALL render a centered spinning indicator
4. THE Loading_State_Component SHALL support configurable number of skeleton rows
5. THE Loading_State_Component SHALL support dark mode styling

### Requirement 11: Dark Mode Toggle

**User Story:** As a user, I want to toggle between light and dark themes, so that I can use the interface comfortably in different lighting conditions.

#### Acceptance Criteria

1. THE System SHALL provide a dark mode toggle button in the topbar
2. WHEN the user clicks the toggle button, THEN THE System SHALL immediately update the UI theme
3. WHEN the user clicks the toggle button, THEN THE System SHALL send an API request to persist the preference
4. IF the API request succeeds, THEN THE System SHALL maintain the new theme state
5. IF the API request fails, THEN THE System SHALL revert the UI to the previous theme and display an error toast
6. THE toggle button SHALL display a moon icon in light mode and a sun icon in dark mode
7. THE toggle button SHALL include an aria-label for accessibility
8. THE System SHALL apply the dark theme by adding the 'dark' class to the document element

### Requirement 12: CDN Fallback Mechanism

**User Story:** As a user, I want the application to work offline or when CDNs are unavailable, so that I can continue working without interruption.

#### Acceptance Criteria

1. WHEN a CDN resource fails to load, THEN THE System SHALL detect the error via onerror handler
2. WHEN a CDN error is detected, THEN THE System SHALL load the corresponding vendored file from web/static/
3. THE System SHALL log CDN fallback activation to the browser console
4. THE System SHALL provide vendored copies of Tailwind CSS and Alpine.js in web/static/
5. THE application SHALL function with full capability using local static files

### Requirement 13: HTMX Integration

**User Story:** As a developer, I want HTMX to work seamlessly with the new UI components, so that partial page updates maintain consistent styling.

#### Acceptance Criteria

1. WHEN HTMX loads a partial HTML fragment, THEN THE System SHALL apply Tailwind CSS classes correctly
2. WHEN HTMX receives an HX-Trigger header with toast data, THEN THE System SHALL display the toast notification
3. WHEN an HTMX request fails, THEN THE System SHALL trigger an htmx:responseError event
4. WHEN an htmx:responseError event occurs, THEN THE System SHALL display an error toast
5. THE System SHALL preserve existing HTMX attributes during partial updates
6. THE System SHALL show loading states during HTMX requests using hx-indicator

### Requirement 14: Accessibility

**User Story:** As a user with accessibility needs, I want the interface to be keyboard navigable and screen reader friendly, so that I can use the application effectively.

#### Acceptance Criteria

1. THE System SHALL provide aria-label attributes for all icon-only buttons
2. THE System SHALL associate labels with all form inputs
3. THE System SHALL use semantic HTML elements (nav, ul, ol) for navigation and lists
4. THE System SHALL ensure all interactive elements are keyboard accessible
5. THE System SHALL provide focus-visible styles for all focusable elements
6. THE System SHALL maintain a minimum contrast ratio of 4.5:1 for text elements
7. THE System SHALL trap focus within modals when open
8. THE System SHALL move focus appropriately when opening and closing modals

### Requirement 15: Responsive Design

**User Story:** As a user on different devices, I want the interface to adapt to my screen size, so that I can use the application on desktop, tablet, and mobile.

#### Acceptance Criteria

1. THE System SHALL display pages correctly at viewport widths of 768px and above
2. THE System SHALL collapse the sidebar automatically on mobile viewports
3. THE System SHALL adapt layouts using responsive Tailwind CSS classes
4. THE System SHALL ensure touch targets are appropriately sized for mobile devices
5. THE System SHALL maintain usability across different screen sizes

### Requirement 16: Component Validation

**User Story:** As a developer, I want component props to be validated, so that invalid configurations don't break the UI.

#### Acceptance Criteria

1. WHEN a component receives an invalid variant, THEN THE System SHALL use a safe default value
2. WHEN a component receives an invalid size, THEN THE System SHALL use a safe default value
3. WHEN a component is missing a required field, THEN THE System SHALL log a warning in development mode
4. THE System SHALL render components with default styling when validation fails
5. THE System SHALL not throw exceptions for invalid component props

### Requirement 17: State Persistence

**User Story:** As a user, I want my UI preferences to persist, so that the interface remembers my settings.

#### Acceptance Criteria

1. THE System SHALL persist dark mode preference to the user profile in the database
2. THE System SHALL persist sidebar collapse state to localStorage
3. WHEN localStorage is unavailable, THEN THE System SHALL use in-memory state for sidebar collapse
4. WHEN localStorage is unavailable, THEN THE System SHALL not display errors to the user
5. THE System SHALL restore dark mode preference from the database on page load
6. THE System SHALL restore sidebar collapse state from localStorage on page load

### Requirement 18: Performance

**User Story:** As a user, I want the interface to load quickly and respond smoothly, so that I have a pleasant experience.

#### Acceptance Criteria

1. THE System SHALL load dependencies before the closing body tag to avoid render blocking
2. THE System SHALL use x-cloak to prevent flash of unstyled content during Alpine.js initialization
3. THE System SHALL return minimal HTML fragments for HTMX partial updates
4. THE System SHALL use CSS transitions for animations, not JavaScript
5. THE System SHALL limit the toast queue to 4 items to prevent memory issues
6. THE System SHALL debounce expensive operations like search filtering

### Requirement 19: Security

**User Story:** As a system administrator, I want the UI enhancements to maintain security best practices, so that the application remains secure.

#### Acceptance Criteria

1. THE System SHALL use nonce-based Content Security Policy for inline scripts
2. THE System SHALL auto-escape HTML in Go templates by default
3. THE System SHALL use x-text instead of x-html for user-generated content in Alpine.js
4. THE System SHALL validate all HTMX responses on the backend
5. THE System SHALL use CSRF tokens for state-changing requests
6. THE System SHALL store session tokens in HTTP-only cookies, not localStorage
7. THE System SHALL use Subresource Integrity (SRI) hashes for CDN resources

### Requirement 20: Error Handling

**User Story:** As a user, I want clear feedback when errors occur, so that I understand what went wrong and can take appropriate action.

#### Acceptance Criteria

1. WHEN a CDN load fails, THEN THE System SHALL load local fallback and log to console
2. WHEN dark mode persistence fails, THEN THE System SHALL revert the UI and display an error toast
3. WHEN uistyle.json is invalid, THEN THE System SHALL prevent startup and log a descriptive error
4. WHEN localStorage is unavailable, THEN THE System SHALL gracefully degrade to in-memory state
5. WHEN an HTMX request fails, THEN THE System SHALL display an error toast
6. WHEN a modal has no focusable elements, THEN THE System SHALL focus the modal container
7. WHEN a command palette search returns no results, THEN THE System SHALL display "No commands found"
8. WHEN the toast queue overflows, THEN THE System SHALL remove the oldest toast gracefully
