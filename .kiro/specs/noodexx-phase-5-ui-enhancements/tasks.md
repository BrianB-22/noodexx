# Implementation Plan: Noodexx Phase 5 UI Enhancements

## Overview

This implementation plan modernizes the Noodexx UI by integrating Tailwind CSS, Alpine.js, and HTMX. The approach maintains the Go template architecture while introducing a centralized theme configuration system (uistyle.json) and per-user dark mode preferences stored in the database. Implementation follows a bottom-up approach: infrastructure first, then components, then page updates, and finally integration testing.

## Tasks

- [x] 1. Set up UIStyle configuration system
  - [x] 1.1 Create UIStyleConfig Go structs and JSON schema
    - Define UIStyleConfig, ColorScheme, ColorPalette, TypographyConfig, SpacingConfig, RadiusConfig, and ShadowConfig structs
    - Add JSON tags for serialization
    - Create in internal/uistyle/config.go
    - _Requirements: 1.1, 1.3, 1.4, 1.5, 1.6, 1.7_

  - [x] 1.2 Implement color palette validation
    - Write validateColorPalette() function to check all shades 50-900 are present
    - Write isValidHexColor() function to validate #RRGGBB format
    - Return descriptive errors with palette name and missing/invalid shade
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 1.3 Implement UIStyle loader with validation
    - Write loadUIStyle() function to read and parse uistyle.json
    - Call validateUIStyle() to validate all seven color palettes
    - Return error on missing file, malformed JSON, or validation failure
    - _Requirements: 1.1, 1.2, 2.5_

  - [ ]* 1.4 Write property test for color palette validation
    - **Property 1: Color Palette Completeness**
    - **Property 2: Color Value Validity**
    - **Validates: Requirements 2.1, 2.2**

  - [x] 1.5 Create default uistyle.json configuration file
    - Create noodexx/uistyle.json with primary, secondary, success, warning, error, info, and surface palettes
    - Include typography with Inter and JetBrains Mono fonts
    - Include spacing, border radius, and shadow definitions
    - _Requirements: 1.3, 1.4, 1.5, 1.6, 1.7_

  - [x] 1.6 Integrate UIStyle loading into application startup
    - Load UIStyle in main.go before starting server
    - Fail application startup if uistyle.json is invalid
    - Store UIStyleConfig in Server struct for template access
    - _Requirements: 1.1, 1.2_

  - [ ]* 1.7 Write unit tests for UIStyle validation
    - Test missing shades return appropriate errors
    - Test invalid hex codes return appropriate errors
    - Test valid configuration passes validation
    - _Requirements: 2.3, 2.4, 2.5_

- [x] 2. Implement database migration for user dark mode
  - [x] 2.1 Add dark_mode column to users table
    - Create migration to add dark_mode BOOLEAN DEFAULT FALSE column
    - Update User struct in internal/store/models.go with DarkMode field
    - _Requirements: 3.1_

  - [x] 2.2 Implement store methods for dark mode preferences
    - Write UpdateUserDarkMode(userID int, darkMode bool) error
    - Write GetUserDarkMode(userID int) (bool, error)
    - Add to internal/store/store.go
    - _Requirements: 3.2, 3.4_

  - [ ]* 2.3 Write unit tests for dark mode store methods
    - Test UpdateUserDarkMode persists correctly
    - Test GetUserDarkMode retrieves correctly
    - Test error handling for invalid user IDs
    - _Requirements: 3.1, 3.2, 3.4_

- [ ] 3. Create API endpoint for user preferences
  - [x] 3.1 Implement POST /api/user/preferences handler
    - Create UpdatePreferencesRequest struct with DarkMode field
    - Extract user ID from session
    - Call store.UpdateUserDarkMode()
    - Return 200 on success, appropriate error codes on failure
    - _Requirements: 3.4, 11.3_

  - [x] 3.2 Add route registration for preferences endpoint
    - Register POST /api/user/preferences in router
    - Apply authentication middleware
    - _Requirements: 3.4_

  - [ ]* 3.3 Write integration test for preferences API
    - Test authenticated user can update dark mode
    - Test unauthenticated request returns 401
    - Test invalid request body returns 400
    - _Requirements: 3.4, 11.3, 11.4, 11.5_

- [ ] 4. Vendor CDN dependencies and implement fallback
  - [x] 4.1 Download and vendor Tailwind CSS and Alpine.js
    - Download Tailwind CSS 3.4.1 to web/static/tailwind.min.js
    - Download Alpine.js 3.13.5 to web/static/alpine.min.js
    - Verify HTMX is already present in web/static/
    - _Requirements: 4.3, 4.4, 12.4_

  - [x] 4.2 Implement CDN fallback mechanism in base.html
    - Add onerror handlers to CDN script tags
    - Load local static file on CDN failure
    - Log fallback activation to console
    - _Requirements: 4.1, 4.2, 4.4, 12.1, 12.2, 12.3, 12.5_

- [ ] 5. Create reusable component templates
  - [x] 5.1 Create button component template
    - Create web/templates/components/button.html
    - Support variants: primary, secondary, danger, ghost
    - Support sizes: sm, md, lg
    - Support loading and disabled states
    - Default invalid variant to primary, invalid size to md
    - Accept additional CSS classes
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9_

  - [ ]* 5.2 Write property test for button component
    - **Property 12: Button Variant Validation**
    - **Property 13: Button Size Validation**
    - **Property 15: Additional CSS Classes**
    - **Validates: Requirements 5.6, 5.7, 5.8, 16.1, 16.2**

  - [x] 5.3 Create card component template
    - Create web/templates/components/card.html
    - Support optional title
    - Support dark mode styling
    - Accept additional CSS classes
    - Render provided content
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 5.4 Create modal component template
    - Create web/templates/components/modal.html
    - Implement backdrop with click-to-close
    - Implement Escape key handler
    - Add aria-modal and role attributes
    - Support title, content, and actions
    - Add enter/leave transitions
    - Implement focus trap logic
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 7.8, 7.9_

  - [x] 5.5 Create toast notification system
    - Create web/templates/components/toast.html with toast-container
    - Implement toastManager Alpine.js component
    - Support variants: success, error, info, warning
    - Implement auto-dismiss with configurable duration
    - Enforce maximum 4 visible toasts
    - Add manual dismiss button
    - Position in top-right corner with stacking
    - Add enter/leave transitions
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.9_

  - [ ]* 5.6 Write property test for toast queue management
    - **Property 17: Toast Queue Limit**
    - **Property 18: Toast Variant Validity**
    - **Validates: Requirements 8.1, 8.3, 18.5**

  - [x] 5.7 Create empty state component template
    - Create web/templates/components/empty-state.html
    - Support icon, heading, and description
    - Support optional action button
    - Center content in container
    - Support dark mode styling
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

  - [x] 5.8 Create loading state component template
    - Create web/templates/components/loading-state.html
    - Support skeleton and spinner types
    - Implement animated skeleton rows with configurable count
    - Implement centered spinner
    - Support dark mode styling
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [ ] 6. Update base.html page shell
  - [x] 6.1 Implement Tailwind CSS CDN loading with inline config
    - Add Tailwind CSS CDN script tag with fallback
    - Inject UIStyle configuration into tailwind.config
    - Add template helper to convert Go structs to JSON
    - Set darkMode: 'class' in Tailwind config
    - _Requirements: 4.1, 4.5, 1.8_

  - [x] 6.2 Implement Alpine.js initialization with dark mode
    - Add Alpine.js CDN script tag with fallback
    - Initialize x-data with darkMode from PageData.DarkMode
    - Bind :class="{ 'dark': darkMode }" to html element
    - _Requirements: 4.2, 4.6, 3.6, 11.8_

  - [x] 6.3 Implement dark mode toggle button in topbar
    - Add toggle button with moon/sun icons
    - Implement toggleDarkMode() function
    - Send POST to /api/user/preferences on toggle
    - Revert UI and show error toast on API failure
    - Add aria-label for accessibility
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7, 14.1_

  - [x] 6.4 Add global toast notification container
    - Include toast-container component in base.html
    - Add event listener for HTMX HX-Trigger toast events
    - _Requirements: 4.7, 8.8, 13.2_

  - [x] 6.5 Update sidebar and topbar with modern styling
    - Apply Tailwind CSS classes to sidebar and topbar
    - Use semantic HTML (nav, ul, ol)
    - Ensure keyboard accessibility
    - Add focus-visible styles
    - _Requirements: 4.8, 14.3, 14.4, 14.5_

  - [x] 6.6 Implement sidebar collapse functionality
    - Add Alpine.js state for sidebar collapse
    - Persist collapse state to localStorage
    - Handle localStorage unavailable gracefully
    - _Requirements: 17.2, 17.3, 17.4, 17.6_

  - [ ]* 6.7 Write property test for dark mode toggle
    - **Property 6: Dark Mode Persistence**
    - **Property 8: Dark Mode UI Update**
    - **Property 9: Dark Mode Error Revert**
    - **Validates: Requirements 3.4, 3.3, 3.5, 11.2, 11.3, 11.5**

- [ ] 7. Update dashboard page
  - [x] 7.1 Refactor dashboard.html with component templates
    - Use button component for "New Chat" action
    - Use card components for metrics grid
    - Use empty-state component when no activity
    - Apply responsive Tailwind classes
    - _Requirements: 15.1, 15.3_

  - [x] 7.2 Ensure dashboard dark mode styling
    - Test all elements render correctly in dark mode
    - Verify contrast ratios meet 4.5:1 minimum
    - _Requirements: 14.6_

- [ ] 8. Update chat page
  - [x] 8.1 Refactor chat.html with component templates
    - Use card components for session list and chat area
    - Use button component for send button
    - Use loading-state component for message loading
    - Implement HTMX partial updates for messages
    - Add hx-indicator for loading states
    - _Requirements: 13.1, 13.6_

  - [x] 8.2 Implement HTMX error handling for chat
    - Add htmx:responseError event listener
    - Display error toast on request failure
    - _Requirements: 13.3, 13.4, 20.5_

  - [x] 8.3 Ensure chat page accessibility
    - Add labels for message input
    - Ensure keyboard navigation works
    - Test with screen reader
    - _Requirements: 14.2, 14.4_

- [ ] 9. Update library page
  - [x] 9.1 Refactor library.html with component templates
    - Use card components for document list
    - Use button components for upload and delete actions
    - Use modal component for delete confirmation
    - Use empty-state component when no documents
    - Use loading-state component during uploads
    - _Requirements: 13.1_

  - [x] 9.2 Implement HTMX toast triggers for library actions
    - Add HX-Trigger headers for upload success/failure
    - Add HX-Trigger headers for delete success/failure
    - _Requirements: 8.8, 13.2_

  - [x] 9.3 Ensure library page responsive design
    - Test at 768px and above
    - Ensure touch targets are appropriately sized
    - _Requirements: 15.1, 15.4_

- [ ] 10. Update settings page
  - [x] 10.1 Refactor settings.html with component templates
    - Use card components for settings sections
    - Use button components for save actions
    - Apply form styling with Tailwind
    - _Requirements: 13.1_

  - [x] 10.2 Ensure settings form accessibility
    - Associate labels with all inputs
    - Add appropriate aria attributes
    - Test keyboard navigation
    - _Requirements: 14.2, 14.4_

- [ ] 11. Update login page
  - [x] 11.1 Refactor login.html with modern styling
    - Apply Tailwind CSS classes to form
    - Use button component for submit button
    - Ensure responsive design
    - _Requirements: 15.1, 15.3_

  - [x] 11.2 Ensure login page accessibility
    - Associate labels with inputs
    - Add focus-visible styles
    - Test keyboard navigation
    - _Requirements: 14.2, 14.5, 14.4_

- [ ] 12. Implement security enhancements
  - [x] 12.1 Add nonce-based CSP for inline scripts
    - Generate unique nonce per request
    - Add nonce to Tailwind config script tag
    - Set Content-Security-Policy header with nonce
    - _Requirements: 19.1_

  - [x] 12.2 Add Subresource Integrity for CDN resources
    - Add integrity and crossorigin attributes to CDN script tags
    - Verify SRI hashes for Tailwind CSS and Alpine.js
    - _Requirements: 19.7_

  - [x] 12.3 Verify template escaping and CSRF tokens
    - Confirm Go templates auto-escape HTML
    - Verify x-text used instead of x-html for user content
    - Confirm CSRF tokens present on state-changing requests
    - _Requirements: 19.2, 19.3, 19.5_

- [x] 13. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 14. Integration testing and validation
  - [ ]* 14.1 Write integration test for dark mode persistence
    - Test toggle updates UI immediately
    - Test preference persists to database
    - Test preference loads on page reload
    - Test preference syncs across sessions
    - **Property 7: Dark Mode Retrieval**
    - **Property 20: Alpine.js Initialization**
    - **Validates: Requirements 3.2, 3.3, 3.4, 3.6, 3.7, 4.6**

  - [ ]* 14.2 Write integration test for CDN fallback
    - Mock CDN failure
    - Verify local static file loads
    - Verify application functions correctly
    - **Property 10: CDN Fallback Detection**
    - **Property 11: CDN Fallback Loading**
    - **Validates: Requirements 12.1, 12.2, 12.5**

  - [ ]* 14.3 Write integration test for HTMX toast triggers
    - Trigger backend action with HX-Trigger header
    - Verify toast displays with correct variant
    - Verify toast auto-dismisses
    - **Property 23: HTMX Toast Trigger**
    - **Validates: Requirements 8.8, 13.2**

  - [ ]* 14.4 Write integration test for modal focus trap
    - Open modal
    - Verify focus moves to first focusable element
    - Tab through elements
    - Verify focus stays within modal
    - Press Escape
    - Verify modal closes
    - **Validates: Requirements 7.2, 7.3, 7.5, 14.7, 14.8**

  - [ ]* 14.5 Write integration test for component validation
    - Pass invalid variant to button component
    - Verify component renders with default styling
    - Verify no exceptions thrown
    - **Property 33: Component Validation Fallback**
    - **Validates: Requirements 16.1, 16.4, 16.5**

  - [ ]* 14.6 Write property test for theme consistency
    - **Property 4: Theme Configuration Injection**
    - **Property 5: Theme Consistency Across Users**
    - **Validates: Requirements 1.8, 1.9**

  - [ ]* 14.7 Write integration test for responsive design
    - Test pages at 768px viewport
    - Verify sidebar collapses on mobile
    - Verify touch targets are appropriately sized
    - **Validates: Requirements 15.1, 15.2, 15.4**

- [x] 15. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- Integration tests validate end-to-end flows across components
- The implementation uses Go for backend, HTML/CSS/JavaScript for frontend
- No build pipeline required - CDN-based approach with local fallbacks
- Dark mode is stored per-user in database, not localStorage
- Theme configuration (uistyle.json) is global and applies to all users
