import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_cs.dart';
import 'app_localizations_de.dart';
import 'app_localizations_en.dart';
import 'app_localizations_es.dart';
import 'app_localizations_eu.dart';
import 'app_localizations_fr.dart';
import 'app_localizations_hu.dart';
import 'app_localizations_it.dart';
import 'app_localizations_nl.dart';
import 'app_localizations_no.dart';
import 'app_localizations_pl.dart';
import 'app_localizations_pt.dart';
import 'app_localizations_ru.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'i18n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('cs'),
    Locale('de'),
    Locale('en'),
    Locale('es'),
    Locale('eu'),
    Locale('fr'),
    Locale('hu'),
    Locale('it'),
    Locale('nl'),
    Locale('no'),
    Locale('pl'),
    Locale('pt'),
    Locale('ru'),
    Locale('zh'),
  ];

  /// Standalone heading used in two places: the 'About' tab on the trail detail panel (about the trail), and prefixed to a username as 'About <username>' on the account screen. Needs to read as a section heading in both.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get about;

  /// No description provided for @appearance.
  ///
  /// In en, this message translates to:
  /// **'Appearance'**
  String get appearance;

  /// No description provided for @account.
  ///
  /// In en, this message translates to:
  /// **'Account'**
  String get account;

  /// No description provided for @account_delete_confirm.
  ///
  /// In en, this message translates to:
  /// **'You are about to delete your account. All your trails will also be deleted. Do you want to proceed?'**
  String get account_delete_confirm;

  /// No description provided for @account_privacy.
  ///
  /// In en, this message translates to:
  /// **'Account privacy'**
  String get account_privacy;

  /// No description provided for @add_bio.
  ///
  /// In en, this message translates to:
  /// **'Add Bio'**
  String get add_bio;

  /// No description provided for @add_waypoint.
  ///
  /// In en, this message translates to:
  /// **'Add Waypoint'**
  String get add_waypoint;

  /// No description provided for @adjust_track.
  ///
  /// In en, this message translates to:
  /// **'Adjust track'**
  String get adjust_track;

  /// Used two ways. (1) Label above the date picker that filters for trails dated after the chosen day. (2) Prefixed to a distance on the waypoint sheet as 'After 2.4 km', meaning how far along the trail the waypoint sits. Pick wording that works for both a date and a distance.
  ///
  /// In en, this message translates to:
  /// **'After'**
  String get after;

  /// Filter chip in global search meaning 'all result categories' (trails, lists, places and users together), as opposed to one specific category.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get all;

  /// No description provided for @altitude.
  ///
  /// In en, this message translates to:
  /// **'Altitude'**
  String get altitude;

  /// Noun. Heading and input placeholder for the filter that narrows trails to a specific author's username.
  ///
  /// In en, this message translates to:
  /// **'Author'**
  String get author;

  /// No description provided for @average_speed.
  ///
  /// In en, this message translates to:
  /// **'Avg. Speed'**
  String get average_speed;

  /// No description provided for @background_location_body.
  ///
  /// In en, this message translates to:
  /// **'wanderer collects location data in the background so your trail keeps recording when the screen is off or the app is closed. Your recorded track stays on your device until you choose to save the trail.\n\nAndroid only offers this in system settings: open Location and choose \"Allow all the time\".'**
  String get background_location_body;

  /// No description provided for @background_location_confirm.
  ///
  /// In en, this message translates to:
  /// **'Open settings'**
  String get background_location_confirm;

  /// No description provided for @background_location_title.
  ///
  /// In en, this message translates to:
  /// **'Keep recording in the background'**
  String get background_location_title;

  /// No description provided for @basic_info.
  ///
  /// In en, this message translates to:
  /// **'Basic Info'**
  String get basic_info;

  /// Label above the date picker that filters for trails dated before the chosen day. Pairs with the 'after' string.
  ///
  /// In en, this message translates to:
  /// **'Before'**
  String get before;

  /// Byline preposition, rendered lowercase and immediately followed by a username: 'by alice'. Appears on trail cards, list cards and summit log entries. Not the standalone word 'by' in any other sense.
  ///
  /// In en, this message translates to:
  /// **'by'**
  String get by;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @discard.
  ///
  /// In en, this message translates to:
  /// **'Discard'**
  String get discard;

  /// No description provided for @discard_trail_confirm.
  ///
  /// In en, this message translates to:
  /// **'Discard this trail and its changes?'**
  String get discard_trail_confirm;

  /// No description provided for @card.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Card} other {Cards}}'**
  String card(num n);

  /// No description provided for @categories.
  ///
  /// In en, this message translates to:
  /// **'Categories'**
  String get categories;

  /// No description provided for @category.
  ///
  /// In en, this message translates to:
  /// **'Category'**
  String get category;

  /// No description provided for @change_email.
  ///
  /// In en, this message translates to:
  /// **'Change email'**
  String get change_email;

  /// No description provided for @change_password.
  ///
  /// In en, this message translates to:
  /// **'Change password'**
  String get change_password;

  /// No description provided for @clear_all.
  ///
  /// In en, this message translates to:
  /// **'Clear all'**
  String get clear_all;

  /// No description provided for @close.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get close;

  /// No description provided for @comment.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Comment} other {Comments}}'**
  String comment(num n);

  /// No description provided for @completed.
  ///
  /// In en, this message translates to:
  /// **'Completed'**
  String get completed;

  /// No description provided for @completion_status.
  ///
  /// In en, this message translates to:
  /// **'Completion Status'**
  String get completion_status;

  /// No description provided for @confirm_deletion.
  ///
  /// In en, this message translates to:
  /// **'Confirm Deletion'**
  String get confirm_deletion;

  /// No description provided for @couldnt_start_navigation.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t start navigation. Check your connection and try again.'**
  String get couldnt_start_navigation;

  /// No description provided for @location_services_disabled.
  ///
  /// In en, this message translates to:
  /// **'Location services are disabled. Please enable GPS to use navigation.'**
  String get location_services_disabled;

  /// No description provided for @location_permission_denied.
  ///
  /// In en, this message translates to:
  /// **'Location permission is required for navigation.'**
  String get location_permission_denied;

  /// No description provided for @location_permission_permanently_denied.
  ///
  /// In en, this message translates to:
  /// **'Location permission is permanently denied. Please enable it in Settings.'**
  String get location_permission_permanently_denied;

  /// No description provided for @location_unavailable.
  ///
  /// In en, this message translates to:
  /// **'Unable to determine your location. Please try again.'**
  String get location_unavailable;

  /// No description provided for @copy_link.
  ///
  /// In en, this message translates to:
  /// **'Copy Link'**
  String get copy_link;

  /// No description provided for @create_waypoint.
  ///
  /// In en, this message translates to:
  /// **'Create waypoint'**
  String get create_waypoint;

  /// No description provided for @creation_date.
  ///
  /// In en, this message translates to:
  /// **'Creation date'**
  String get creation_date;

  /// No description provided for @current_password.
  ///
  /// In en, this message translates to:
  /// **'Current password'**
  String get current_password;

  /// No description provided for @danger_zone.
  ///
  /// In en, this message translates to:
  /// **'Danger zone'**
  String get danger_zone;

  /// Noun. Used as a sort option ('sort trails by date'), as the heading of the date-range filter sheet, and as the label of the date field in the trail form. A single neutral noun is needed for all three.
  ///
  /// In en, this message translates to:
  /// **'Date'**
  String get date;

  /// No description provided for @delete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get delete;

  /// No description provided for @not_now.
  ///
  /// In en, this message translates to:
  /// **'Not now'**
  String get not_now;

  /// Verb. Menu action that opens a downloaded trail from the library. Not the adjective 'open' and not the opposite of 'closed'.
  ///
  /// In en, this message translates to:
  /// **'Open'**
  String get open;

  /// No description provided for @delete_account.
  ///
  /// In en, this message translates to:
  /// **'Delete Account'**
  String get delete_account;

  /// No description provided for @delete_trail_confirm.
  ///
  /// In en, this message translates to:
  /// **'Do you really want to delete this trail? This action cannot be undone.'**
  String get delete_trail_confirm;

  /// No description provided for @delete_blocked_while_uploading.
  ///
  /// In en, this message translates to:
  /// **'This trail is uploading right now. Wait for the upload to finish, then try again.'**
  String get delete_blocked_while_uploading;

  /// No description provided for @delete_unsynced_trail_confirm.
  ///
  /// In en, this message translates to:
  /// **'Delete this trail? It hasn\'t been uploaded yet, so this can\'t be undone.'**
  String get delete_unsynced_trail_confirm;

  /// No description provided for @delete_needs_connection.
  ///
  /// In en, this message translates to:
  /// **'This trail is already on the server. Connect to the internet to delete it.'**
  String get delete_needs_connection;

  /// No description provided for @description.
  ///
  /// In en, this message translates to:
  /// **'Description'**
  String get description;

  /// No description provided for @difficult.
  ///
  /// In en, this message translates to:
  /// **'Difficult'**
  String get difficult;

  /// No description provided for @difficulty.
  ///
  /// In en, this message translates to:
  /// **'Difficulty'**
  String get difficulty;

  /// No description provided for @directions.
  ///
  /// In en, this message translates to:
  /// **'Directions'**
  String get directions;

  /// No description provided for @distance.
  ///
  /// In en, this message translates to:
  /// **'Distance'**
  String get distance;

  /// No description provided for @download.
  ///
  /// In en, this message translates to:
  /// **'Download'**
  String get download;

  /// No description provided for @duration.
  ///
  /// In en, this message translates to:
  /// **'Duration'**
  String get duration;

  /// No description provided for @easy.
  ///
  /// In en, this message translates to:
  /// **'Easy'**
  String get easy;

  /// No description provided for @edit.
  ///
  /// In en, this message translates to:
  /// **'Edit'**
  String get edit;

  /// No description provided for @edit_needs_connection.
  ///
  /// In en, this message translates to:
  /// **'Editing works on the server copy of this trail. Connect to the internet to edit it.'**
  String get edit_needs_connection;

  /// No description provided for @edit_waypoint.
  ///
  /// In en, this message translates to:
  /// **'Edit Waypoint'**
  String get edit_waypoint;

  /// Adjective appended in parentheses after a comment's timestamp to mark that the comment was changed after posting: '2 hours ago (edited)'. Rendered lowercase.
  ///
  /// In en, this message translates to:
  /// **'edited'**
  String get edited;

  /// No description provided for @elevation_gain.
  ///
  /// In en, this message translates to:
  /// **'Elevation Gain'**
  String get elevation_gain;

  /// No description provided for @elevation_loss.
  ///
  /// In en, this message translates to:
  /// **'Elevation Loss'**
  String get elevation_loss;

  /// No description provided for @elevation_profile.
  ///
  /// In en, this message translates to:
  /// **'Elevation Profile'**
  String get elevation_profile;

  /// No description provided for @email.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get email;

  /// No description provided for @email_not_unique.
  ///
  /// In en, this message translates to:
  /// **'This email address is already in use.'**
  String get email_not_unique;

  /// No description provided for @email_updated.
  ///
  /// In en, this message translates to:
  /// **'Email updated'**
  String get email_updated;

  /// No description provided for @error_deleting_trail.
  ///
  /// In en, this message translates to:
  /// **'Error deleting trail'**
  String get error_deleting_trail;

  /// No description provided for @error_reading_file.
  ///
  /// In en, this message translates to:
  /// **'Error reading file'**
  String get error_reading_file;

  /// No description provided for @error_saving_settings.
  ///
  /// In en, this message translates to:
  /// **'Error saving settings'**
  String get error_saving_settings;

  /// No description provided for @error_saving_trail.
  ///
  /// In en, this message translates to:
  /// **'Error saving trail'**
  String get error_saving_trail;

  /// No description provided for @error_updating_password.
  ///
  /// In en, this message translates to:
  /// **'Error updating password'**
  String get error_updating_password;

  /// No description provided for @explore.
  ///
  /// In en, this message translates to:
  /// **'Explore'**
  String get explore;

  /// No description provided for @exit_navigation.
  ///
  /// In en, this message translates to:
  /// **'Exit'**
  String get exit_navigation;

  /// No description provided for @stop_navigation_confirm.
  ///
  /// In en, this message translates to:
  /// **'Stop navigation and return to the trail?'**
  String get stop_navigation_confirm;

  /// No description provided for @stop_recording.
  ///
  /// In en, this message translates to:
  /// **'Stop recording'**
  String get stop_recording;

  /// No description provided for @stop_recording_confirm.
  ///
  /// In en, this message translates to:
  /// **'Stop recording?'**
  String get stop_recording_confirm;

  /// No description provided for @search_this_area.
  ///
  /// In en, this message translates to:
  /// **'Search this area'**
  String get search_this_area;

  /// No description provided for @filter_tags.
  ///
  /// In en, this message translates to:
  /// **'Filter tags'**
  String get filter_tags;

  /// No description provided for @filter_trails.
  ///
  /// In en, this message translates to:
  /// **'Filter trails'**
  String get filter_trails;

  /// Used two ways. (1) Noun: label of the last row in the trail's waypoint timeline, marking the end of the trail. (2) Tooltip on the route planner's confirm action, meaning 'finish planning this route'. Prefer wording that carries both, or the noun sense if it must be one.
  ///
  /// In en, this message translates to:
  /// **'Finish'**
  String get finish;

  /// No description provided for @finish_disabled_hint.
  ///
  /// In en, this message translates to:
  /// **'Add at least 2 anchors to finish your route.'**
  String get finish_disabled_hint;

  /// No description provided for @follow.
  ///
  /// In en, this message translates to:
  /// **'Follow'**
  String get follow;

  /// No description provided for @followers.
  ///
  /// In en, this message translates to:
  /// **'Followers'**
  String get followers;

  /// No description provided for @following.
  ///
  /// In en, this message translates to:
  /// **'Following'**
  String get following;

  /// No description provided for @from_photos.
  ///
  /// In en, this message translates to:
  /// **'From Photos'**
  String get from_photos;

  /// No description provided for @help.
  ///
  /// In en, this message translates to:
  /// **'Help'**
  String get help;

  /// No description provided for @hiking.
  ///
  /// In en, this message translates to:
  /// **'Hiking'**
  String get hiking;

  /// No description provided for @home.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get home;

  /// Noun. Label of the icon picker field in the waypoint form — which symbol to show for this waypoint on the map.
  ///
  /// In en, this message translates to:
  /// **'Icon'**
  String get icon;

  /// No description provided for @imperial.
  ///
  /// In en, this message translates to:
  /// **'Imperial'**
  String get imperial;

  /// No description provided for @joined.
  ///
  /// In en, this message translates to:
  /// **'Joined'**
  String get joined;

  /// No description provided for @language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// No description provided for @latitude.
  ///
  /// In en, this message translates to:
  /// **'Latitude'**
  String get latitude;

  /// No description provided for @like_status.
  ///
  /// In en, this message translates to:
  /// **'Like Status'**
  String get like_status;

  /// Adjective. Filter toggle that narrows the list to trails you have liked.
  ///
  /// In en, this message translates to:
  /// **'Liked'**
  String get liked;

  /// No description provided for @likes.
  ///
  /// In en, this message translates to:
  /// **'Likes'**
  String get likes;

  /// No description provided for @list.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {List} other {Lists}}'**
  String list(num n);

  /// No description provided for @location.
  ///
  /// In en, this message translates to:
  /// **'Location'**
  String get location;

  /// No description provided for @locations.
  ///
  /// In en, this message translates to:
  /// **'Locations'**
  String get locations;

  /// No description provided for @center_on_my_location.
  ///
  /// In en, this message translates to:
  /// **'Center on my location'**
  String get center_on_my_location;

  /// No description provided for @location_tracking_notification_title.
  ///
  /// In en, this message translates to:
  /// **'wanderer'**
  String get location_tracking_notification_title;

  /// No description provided for @location_tracking_notification_text.
  ///
  /// In en, this message translates to:
  /// **'Recording your trail'**
  String get location_tracking_notification_text;

  /// No description provided for @location_tracking_notification_text_navigating.
  ///
  /// In en, this message translates to:
  /// **'Navigating {trail}'**
  String location_tracking_notification_text_navigating(String trail);

  /// No description provided for @login.
  ///
  /// In en, this message translates to:
  /// **'Login'**
  String get login;

  /// No description provided for @logout.
  ///
  /// In en, this message translates to:
  /// **'Logout'**
  String get logout;

  /// No description provided for @longitude.
  ///
  /// In en, this message translates to:
  /// **'Longitude'**
  String get longitude;

  /// No description provided for @map.
  ///
  /// In en, this message translates to:
  /// **'Map'**
  String get map;

  /// No description provided for @metric.
  ///
  /// In en, this message translates to:
  /// **'Metric'**
  String get metric;

  /// No description provided for @moderate.
  ///
  /// In en, this message translates to:
  /// **'Moderate'**
  String get moderate;

  /// No description provided for @my_account.
  ///
  /// In en, this message translates to:
  /// **'My Account'**
  String get my_account;

  /// No description provided for @in_distance.
  ///
  /// In en, this message translates to:
  /// **'in {distance}'**
  String in_distance(String distance);

  /// Noun. Used as a sort option ('sort by name') and as the label of the trail name field in the trail form.
  ///
  /// In en, this message translates to:
  /// **'Name'**
  String get name;

  /// No description provided for @navigate.
  ///
  /// In en, this message translates to:
  /// **'Navigate'**
  String get navigate;

  /// No description provided for @new_password.
  ///
  /// In en, this message translates to:
  /// **'New password'**
  String get new_password;

  /// No description provided for @new_password_success.
  ///
  /// In en, this message translates to:
  /// **'The new password has been set'**
  String get new_password_success;

  /// No description provided for @new_trail.
  ///
  /// In en, this message translates to:
  /// **'New Trail'**
  String get new_trail;

  /// No description provided for @trail_source_planner.
  ///
  /// In en, this message translates to:
  /// **'Open trail planner'**
  String get trail_source_planner;

  /// No description provided for @trail_source_planner_description.
  ///
  /// In en, this message translates to:
  /// **'Draw a new route on the map, waypoint by waypoint.'**
  String get trail_source_planner_description;

  /// No description provided for @trail_source_record.
  ///
  /// In en, this message translates to:
  /// **'Record trail'**
  String get trail_source_record;

  /// No description provided for @trail_source_record_description.
  ///
  /// In en, this message translates to:
  /// **'Track your live coordinates and log your journey in real-time.'**
  String get trail_source_record_description;

  /// No description provided for @trail_source_import.
  ///
  /// In en, this message translates to:
  /// **'Import file'**
  String get trail_source_import;

  /// No description provided for @trail_source_import_description.
  ///
  /// In en, this message translates to:
  /// **'Upload GPX, KML, KMZ, TCX or FIT files directly from your device storage.'**
  String get trail_source_import_description;

  /// No description provided for @trail_source_import_error.
  ///
  /// In en, this message translates to:
  /// **'Could not import file'**
  String get trail_source_import_error;

  /// Shown when the user picks a non-GPX file for import while offline. Names the formats that need a connection and points at GPX as the offline-capable alternative.
  ///
  /// In en, this message translates to:
  /// **'Only GPX files can be imported offline'**
  String get trail_source_offline_import_error;

  /// No description provided for @no_comments_so_far.
  ///
  /// In en, this message translates to:
  /// **'No comments so far'**
  String get no_comments_so_far;

  /// No description provided for @no_data.
  ///
  /// In en, this message translates to:
  /// **'No data'**
  String get no_data;

  /// No description provided for @no_description_for_now.
  ///
  /// In en, this message translates to:
  /// **'No description for now'**
  String get no_description_for_now;

  /// No description provided for @no_gps_data_in_image.
  ///
  /// In en, this message translates to:
  /// **'No GPS data found in this photo.'**
  String get no_gps_data_in_image;

  /// No description provided for @no_preference.
  ///
  /// In en, this message translates to:
  /// **'No preference'**
  String get no_preference;

  /// No description provided for @no_trails_found.
  ///
  /// In en, this message translates to:
  /// **'No trails found'**
  String get no_trails_found;

  /// No description provided for @not_completed.
  ///
  /// In en, this message translates to:
  /// **'Not completed'**
  String get not_completed;

  /// No description provided for @notifications.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get notifications;

  /// No description provided for @only_me.
  ///
  /// In en, this message translates to:
  /// **'Only me'**
  String get only_me;

  /// Divider word between the email login form and the OAuth provider buttons on the login screen.
  ///
  /// In en, this message translates to:
  /// **'or'**
  String get or;

  /// No description provided for @own_trails_empty_body.
  ///
  /// In en, this message translates to:
  /// **'Trails you record or save offline appear here, and upload automatically once you\'re back online.'**
  String get own_trails_empty_body;

  /// No description provided for @own_trails_empty_title.
  ///
  /// In en, this message translates to:
  /// **'Nothing saved yet'**
  String get own_trails_empty_title;

  /// No description provided for @own_trails_offline_banner.
  ///
  /// In en, this message translates to:
  /// **'Offline — showing only trails on this device.'**
  String get own_trails_offline_banner;

  /// Label of the profile trail count card when the count comes from local storage (unsynced captures plus downloaded trails) because the device is offline.
  ///
  /// In en, this message translates to:
  /// **'Trails (on device)'**
  String get trails_on_device;

  /// No description provided for @pause.
  ///
  /// In en, this message translates to:
  /// **'Pause'**
  String get pause;

  /// No description provided for @password.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get password;

  /// No description provided for @password_confirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm password'**
  String get password_confirm;

  /// No description provided for @passwords_must_match.
  ///
  /// In en, this message translates to:
  /// **'Passwords must match'**
  String get passwords_must_match;

  /// No description provided for @photo_copy_failed_toast.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1 {Trail saved, but 1 photo couldn\'t be saved.} other {Trail saved, but {count} photos couldn\'t be saved.}}'**
  String photo_copy_failed_toast(num count);

  /// No description provided for @photos.
  ///
  /// In en, this message translates to:
  /// **'Photos & Videos'**
  String get photos;

  /// No description provided for @photos_skipped_no_gps.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1 {1 photo skipped — no GPS data} other {{count} photos skipped — no GPS data}}'**
  String photos_skipped_no_gps(num count);

  /// No description provided for @privacy.
  ///
  /// In en, this message translates to:
  /// **'Privacy'**
  String get privacy;

  /// No description provided for @private.
  ///
  /// In en, this message translates to:
  /// **'Private'**
  String get private;

  /// No description provided for @profile.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get profile;

  /// No description provided for @public.
  ///
  /// In en, this message translates to:
  /// **'Public'**
  String get public;

  /// No description provided for @reached_end_of_trail.
  ///
  /// In en, this message translates to:
  /// **'You\'ve reached the end of the trail.'**
  String get reached_end_of_trail;

  /// No description provided for @register.
  ///
  /// In en, this message translates to:
  /// **'Register'**
  String get register;

  /// No description provided for @reorder_photos_hint.
  ///
  /// In en, this message translates to:
  /// **'Long-press and drag to reorder photos.'**
  String get reorder_photos_hint;

  /// Verb. Button that clears every active filter back to its default. Not 'restart' and not 'reset password'.
  ///
  /// In en, this message translates to:
  /// **'Reset'**
  String get reset;

  /// No description provided for @resume.
  ///
  /// In en, this message translates to:
  /// **'Resume'**
  String get resume;

  /// No description provided for @resume_navigation_prompt.
  ///
  /// In en, this message translates to:
  /// **'Resume navigation on {trail}?'**
  String resume_navigation_prompt(String trail);

  /// No description provided for @resume_recording_prompt.
  ///
  /// In en, this message translates to:
  /// **'Resume recording?'**
  String get resume_recording_prompt;

  /// No description provided for @route.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Route} other {Routes}}'**
  String route(num n);

  /// No description provided for @follow_roads.
  ///
  /// In en, this message translates to:
  /// **'Follow roads'**
  String get follow_roads;

  /// No description provided for @follow_roads_description.
  ///
  /// In en, this message translates to:
  /// **'Snap the recorded path to the nearest roads and trails.'**
  String get follow_roads_description;

  /// No description provided for @recalculate_heights.
  ///
  /// In en, this message translates to:
  /// **'Recalculate heights'**
  String get recalculate_heights;

  /// No description provided for @recalculate_heights_description.
  ///
  /// In en, this message translates to:
  /// **'Replace recorded GPS elevation with more accurate values from the map.'**
  String get recalculate_heights_description;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @save_recording_options.
  ///
  /// In en, this message translates to:
  /// **'Save recording'**
  String get save_recording_options;

  /// No description provided for @save_track.
  ///
  /// In en, this message translates to:
  /// **'Save track'**
  String get save_track;

  /// No description provided for @search.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get search;

  /// No description provided for @search_for_trails_places.
  ///
  /// In en, this message translates to:
  /// **'Search for trails, lists, places'**
  String get search_for_trails_places;

  /// No description provided for @select_date.
  ///
  /// In en, this message translates to:
  /// **'Select date'**
  String get select_date;

  /// No description provided for @settings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settings;

  /// No description provided for @settings_notification_comment_mention.
  ///
  /// In en, this message translates to:
  /// **'Someone mentioned you in a comment'**
  String get settings_notification_comment_mention;

  /// No description provided for @settings_notification_list_share.
  ///
  /// In en, this message translates to:
  /// **'Someone shared a list with you'**
  String get settings_notification_list_share;

  /// No description provided for @settings_notification_new_follower.
  ///
  /// In en, this message translates to:
  /// **'You have a new follower'**
  String get settings_notification_new_follower;

  /// No description provided for @settings_notification_summit_log_create.
  ///
  /// In en, this message translates to:
  /// **'Someone created a summit log on your trail'**
  String get settings_notification_summit_log_create;

  /// No description provided for @settings_notification_summit_log_mention.
  ///
  /// In en, this message translates to:
  /// **'Someone mentioned you in a summit log'**
  String get settings_notification_summit_log_mention;

  /// No description provided for @settings_notification_trail_comment.
  ///
  /// In en, this message translates to:
  /// **'Someone left a comment on your trail'**
  String get settings_notification_trail_comment;

  /// No description provided for @settings_notification_trail_like.
  ///
  /// In en, this message translates to:
  /// **'Someone liked your trail'**
  String get settings_notification_trail_like;

  /// No description provided for @settings_notification_trail_mention.
  ///
  /// In en, this message translates to:
  /// **'Someone mentioned you in a trail'**
  String get settings_notification_trail_mention;

  /// No description provided for @settings_notification_trail_share.
  ///
  /// In en, this message translates to:
  /// **'Someone shared a trail with you'**
  String get settings_notification_trail_share;

  /// No description provided for @settings_privacy_account_private.
  ///
  /// In en, this message translates to:
  /// **'Only you can see your profile. You will not appear in search results. Other users cannot follow you or share trails with you. You can still publish trails or lists.'**
  String get settings_privacy_account_private;

  /// No description provided for @settings_privacy_account_public.
  ///
  /// In en, this message translates to:
  /// **'Everyone can see your profile. You appear in search results. Other users can follow you and share trails with you.'**
  String get settings_privacy_account_public;

  /// No description provided for @settings_privacy_lists_private.
  ///
  /// In en, this message translates to:
  /// **'Your lists are private by default. No one except you will be able to see them. You can change this setting at any point for individual lists.'**
  String get settings_privacy_lists_private;

  /// No description provided for @settings_privacy_lists_public.
  ///
  /// In en, this message translates to:
  /// **'Your lists are public by default. Everyone will be able to see them. You can change this setting at any point for individual lists.'**
  String get settings_privacy_lists_public;

  /// No description provided for @settings_privacy_trails_private.
  ///
  /// In en, this message translates to:
  /// **'Your trails are private by default. No one except you will be able to see them. You can change this setting at any point for individual trails.'**
  String get settings_privacy_trails_private;

  /// No description provided for @settings_privacy_trails_public.
  ///
  /// In en, this message translates to:
  /// **'Your trails are public by default. Everyone will be able to see them. You can change this setting at any point for individual trails.'**
  String get settings_privacy_trails_public;

  /// No description provided for @share.
  ///
  /// In en, this message translates to:
  /// **'Share'**
  String get share;

  /// No description provided for @share_profile.
  ///
  /// In en, this message translates to:
  /// **'Share profile'**
  String get share_profile;

  /// Adjective. Filter toggle that narrows the list to trails other users have shared with you.
  ///
  /// In en, this message translates to:
  /// **'Shared'**
  String get shared;

  /// No description provided for @show_on_map.
  ///
  /// In en, this message translates to:
  /// **'Show on map'**
  String get show_on_map;

  /// No description provided for @signout_unsynced_warning.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1 {You have 1 trail not uploaded yet. Signing out won\'t delete it — it\'ll be here when you sign back in — but it won\'t upload until then.} other {You have {count} trails not uploaded yet. Signing out won\'t delete them — they\'ll be here when you sign back in — but they won\'t upload until then.}}'**
  String signout_unsynced_warning(num count);

  /// No description provided for @slogan.
  ///
  /// In en, this message translates to:
  /// **'Your trails. Your data. Your server.'**
  String get slogan;

  /// Button label that opens the sort options for a trail or list collection. Verb or noun, whichever reads better as a short button.
  ///
  /// In en, this message translates to:
  /// **'Sort'**
  String get sort;

  /// Noun. Label of the current-speed readout on the live navigation screen.
  ///
  /// In en, this message translates to:
  /// **'Speed'**
  String get speed;

  /// Noun. Label of the first row in the trail's waypoint timeline, marking the trailhead. Not the verb 'to start'.
  ///
  /// In en, this message translates to:
  /// **'Start'**
  String get start;

  /// Label for the subcategories filter section
  ///
  /// In en, this message translates to:
  /// **'Subcategories'**
  String get subcategories;

  /// No description provided for @summit_book.
  ///
  /// In en, this message translates to:
  /// **'Summit Book'**
  String get summit_book;

  /// No description provided for @sync_failed.
  ///
  /// In en, this message translates to:
  /// **'Upload failed · Tap to retry'**
  String get sync_failed;

  /// No description provided for @sync_pending.
  ///
  /// In en, this message translates to:
  /// **'Waiting to upload'**
  String get sync_pending;

  /// No description provided for @sync_uploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading…'**
  String get sync_uploading;

  /// No description provided for @tags.
  ///
  /// In en, this message translates to:
  /// **'Tags'**
  String get tags;

  /// No description provided for @theme_dark.
  ///
  /// In en, this message translates to:
  /// **'Dark'**
  String get theme_dark;

  /// No description provided for @theme_light.
  ///
  /// In en, this message translates to:
  /// **'Light'**
  String get theme_light;

  /// No description provided for @theme_system.
  ///
  /// In en, this message translates to:
  /// **'Follow system'**
  String get theme_system;

  /// No description provided for @time_in_motion.
  ///
  /// In en, this message translates to:
  /// **'Time in Motion'**
  String get time_in_motion;

  /// No description provided for @trail.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Trail} other {Trails}}'**
  String trail(num n);

  /// No description provided for @trail_saved_successfully.
  ///
  /// In en, this message translates to:
  /// **'Trail saved successfully'**
  String get trail_saved_successfully;

  /// No description provided for @trail_not_on_this_device.
  ///
  /// In en, this message translates to:
  /// **'This trail is no longer on this device.'**
  String get trail_not_on_this_device;

  /// No description provided for @trail_uploaded_reopen_to_edit.
  ///
  /// In en, this message translates to:
  /// **'This trail finished uploading. Re-open it from your trails to keep editing.'**
  String get trail_uploaded_reopen_to_edit;

  /// No description provided for @some_waypoints_failed_to_save.
  ///
  /// In en, this message translates to:
  /// **'Trail saved, but some waypoints failed to save'**
  String get some_waypoints_failed_to_save;

  /// No description provided for @units.
  ///
  /// In en, this message translates to:
  /// **'Units'**
  String get units;

  /// No description provided for @users.
  ///
  /// In en, this message translates to:
  /// **'Users'**
  String get users;

  /// No description provided for @username.
  ///
  /// In en, this message translates to:
  /// **'Username'**
  String get username;

  /// No description provided for @username_not_unique.
  ///
  /// In en, this message translates to:
  /// **'This username is already taken. Please try another.'**
  String get username_not_unique;

  /// No description provided for @visibilty_status.
  ///
  /// In en, this message translates to:
  /// **'Visibility status'**
  String get visibilty_status;

  /// Noun naming a notification delivery channel: notifications shown in the web app, as opposed to push notifications on the phone. Appears as a toggle label under each notification type.
  ///
  /// In en, this message translates to:
  /// **'Web'**
  String get web;

  /// No description provided for @waypoints.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Waypoint} other {Waypoints}}'**
  String waypoints(num n);

  /// Sentence fragment on the welcome screen, followed by the untranslated product name: 'Welcome to wanderer'. Keep it as a fragment; the brand name is appended in code and must not be included in the translation.
  ///
  /// In en, this message translates to:
  /// **'Welcome to'**
  String get welcome_to;

  /// No description provided for @wrong_username_or_password.
  ///
  /// In en, this message translates to:
  /// **'Wrong username or password'**
  String get wrong_username_or_password;

  /// No description provided for @you_have_arrived.
  ///
  /// In en, this message translates to:
  /// **'You\'ve arrived'**
  String get you_have_arrived;

  /// No description provided for @settings_categories_confirm_disable_title.
  ///
  /// In en, this message translates to:
  /// **'Hide this category?'**
  String get settings_categories_confirm_disable_title;

  /// No description provided for @settings_categories_confirm_disable_subcategory_title.
  ///
  /// In en, this message translates to:
  /// **'Hide this subcategory?'**
  String get settings_categories_confirm_disable_subcategory_title;

  /// No description provided for @settings_categories_confirm_disable_body.
  ///
  /// In en, this message translates to:
  /// **'{count} of your trails use this category. They will stay published but this filter will be hidden.'**
  String settings_categories_confirm_disable_body(int count);

  /// No description provided for @settings_categories_confirm_view_trails.
  ///
  /// In en, this message translates to:
  /// **'View trails'**
  String get settings_categories_confirm_view_trails;

  /// No description provided for @settings_categories_confirm_disable_confirm.
  ///
  /// In en, this message translates to:
  /// **'Disable anyway'**
  String get settings_categories_confirm_disable_confirm;

  /// No description provided for @settings_categories_empty_title.
  ///
  /// In en, this message translates to:
  /// **'No subcategories'**
  String get settings_categories_empty_title;

  /// No description provided for @settings_categories_empty_body.
  ///
  /// In en, this message translates to:
  /// **'This category has no subcategories to configure.'**
  String get settings_categories_empty_body;

  /// No description provided for @settings_categories_reorder_hint.
  ///
  /// In en, this message translates to:
  /// **'Categories control which trail types you see and in what order. Turn one off to hide it as a filter — your trails stay published, they just won\'t appear under that category. Tap a category to manage its subcategories individually.\n\nTo change the order, press and hold a row, then drag it to a new position. The order you set here is reflected everywhere categories are shown.'**
  String get settings_categories_reorder_hint;

  /// No description provided for @something_went_wrong.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong'**
  String get something_went_wrong;

  /// No description provided for @technical_details.
  ///
  /// In en, this message translates to:
  /// **'Technical Details'**
  String get technical_details;

  /// No description provided for @link.
  ///
  /// In en, this message translates to:
  /// **'Link'**
  String get link;

  /// No description provided for @url.
  ///
  /// In en, this message translates to:
  /// **'URL'**
  String get url;

  /// No description provided for @open_in_new_tab.
  ///
  /// In en, this message translates to:
  /// **'Open in new tab'**
  String get open_in_new_tab;

  /// No description provided for @remove.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get remove;

  /// No description provided for @remove_download_confirm_body.
  ///
  /// In en, this message translates to:
  /// **'This removes the downloaded copy from this device. The trail itself is not deleted — you\'ll need to download it again to use it offline.'**
  String get remove_download_confirm_body;

  /// Verb. Confirm button in the rich text editor's link dialog — applies the entered link to the selected text.
  ///
  /// In en, this message translates to:
  /// **'Apply'**
  String get apply;

  /// No description provided for @add_at_least_2_anchors_hint.
  ///
  /// In en, this message translates to:
  /// **'Add at least 2 anchors to see the elevation profile.'**
  String get add_at_least_2_anchors_hint;

  /// No description provided for @reverse_direction.
  ///
  /// In en, this message translates to:
  /// **'Reverse direction'**
  String get reverse_direction;

  /// No description provided for @delete_all.
  ///
  /// In en, this message translates to:
  /// **'Delete all'**
  String get delete_all;

  /// No description provided for @auto_routing.
  ///
  /// In en, this message translates to:
  /// **'Auto-routing'**
  String get auto_routing;

  /// No description provided for @auto_routing_hint.
  ///
  /// In en, this message translates to:
  /// **'Automatically follow roads and paths between anchors.'**
  String get auto_routing_hint;

  /// No description provided for @travel_profile.
  ///
  /// In en, this message translates to:
  /// **'Travel profile'**
  String get travel_profile;

  /// No description provided for @no_track_data.
  ///
  /// In en, this message translates to:
  /// **'No track data'**
  String get no_track_data;

  /// No description provided for @available_offline.
  ///
  /// In en, this message translates to:
  /// **'Available offline'**
  String get available_offline;

  /// No description provided for @no_lists_found.
  ///
  /// In en, this message translates to:
  /// **'No lists found'**
  String get no_lists_found;

  /// No description provided for @search_lists.
  ///
  /// In en, this message translates to:
  /// **'Search lists…'**
  String get search_lists;

  /// No description provided for @search_for_a_location.
  ///
  /// In en, this message translates to:
  /// **'Search for a location'**
  String get search_for_a_location;

  /// No description provided for @no_results_for_query.
  ///
  /// In en, this message translates to:
  /// **'No results for \"{query}\"'**
  String no_results_for_query(String query);

  /// Button label that opens the filter options for a trail collection. Verb or noun, whichever reads better as a short button.
  ///
  /// In en, this message translates to:
  /// **'Filter'**
  String get filter;

  /// No description provided for @no_label_yet.
  ///
  /// In en, this message translates to:
  /// **'No {label} yet.'**
  String no_label_yet(String label);

  /// No description provided for @no_lists_yet.
  ///
  /// In en, this message translates to:
  /// **'No lists yet.'**
  String get no_lists_yet;

  /// No description provided for @no_bio_yet.
  ///
  /// In en, this message translates to:
  /// **'No bio yet.'**
  String get no_bio_yet;

  /// Button label that expands a truncated profile bio to its full text
  ///
  /// In en, this message translates to:
  /// **'Show more'**
  String get show_more;

  /// Button label that collapses an expanded profile bio back to its truncated form
  ///
  /// In en, this message translates to:
  /// **'Show less'**
  String get show_less;

  /// No description provided for @feed.
  ///
  /// In en, this message translates to:
  /// **'Feed'**
  String get feed;

  /// No description provided for @no_trails_yet.
  ///
  /// In en, this message translates to:
  /// **'No trails yet.'**
  String get no_trails_yet;

  /// No description provided for @library_empty_title.
  ///
  /// In en, this message translates to:
  /// **'No downloaded trails'**
  String get library_empty_title;

  /// No description provided for @library_empty_body.
  ///
  /// In en, this message translates to:
  /// **'Trails you download are kept here so you can open them offline.'**
  String get library_empty_body;

  /// No description provided for @library_empty_search_body.
  ///
  /// In en, this message translates to:
  /// **'Try a different search term or clear your filters.'**
  String get library_empty_search_body;

  /// No description provided for @search_location.
  ///
  /// In en, this message translates to:
  /// **'Search location'**
  String get search_location;

  /// No description provided for @no_servers_match_query.
  ///
  /// In en, this message translates to:
  /// **'No servers match \"{query}\"'**
  String no_servers_match_query(String query);

  /// No description provided for @use_custom_url_instead.
  ///
  /// In en, this message translates to:
  /// **'Use custom URL instead'**
  String get use_custom_url_instead;

  /// No description provided for @select_instance.
  ///
  /// In en, this message translates to:
  /// **'Select Instance'**
  String get select_instance;

  /// No description provided for @enter_server_url_hint.
  ///
  /// In en, this message translates to:
  /// **'Enter server URL (e.g. wanderer.to)'**
  String get enter_server_url_hint;

  /// No description provided for @search_library.
  ///
  /// In en, this message translates to:
  /// **'Search library…'**
  String get search_library;

  /// No description provided for @language_and_units.
  ///
  /// In en, this message translates to:
  /// **'{language} & {units}'**
  String language_and_units(String language, String units);

  /// No description provided for @edit_route.
  ///
  /// In en, this message translates to:
  /// **'Edit route'**
  String get edit_route;

  /// No description provided for @undo.
  ///
  /// In en, this message translates to:
  /// **'Undo'**
  String get undo;

  /// No description provided for @redo.
  ///
  /// In en, this message translates to:
  /// **'Redo'**
  String get redo;

  /// No description provided for @bold.
  ///
  /// In en, this message translates to:
  /// **'Bold'**
  String get bold;

  /// No description provided for @italic.
  ///
  /// In en, this message translates to:
  /// **'Italic'**
  String get italic;

  /// No description provided for @underline.
  ///
  /// In en, this message translates to:
  /// **'Underline'**
  String get underline;

  /// No description provided for @bullet_list.
  ///
  /// In en, this message translates to:
  /// **'Bullet list'**
  String get bullet_list;

  /// No description provided for @ordered_list.
  ///
  /// In en, this message translates to:
  /// **'Ordered list'**
  String get ordered_list;

  /// No description provided for @blockquote.
  ///
  /// In en, this message translates to:
  /// **'Blockquote'**
  String get blockquote;

  /// No description provided for @library.
  ///
  /// In en, this message translates to:
  /// **'Library'**
  String get library;

  /// No description provided for @settings_offline_regions_title.
  ///
  /// In en, this message translates to:
  /// **'Offline Maps/Regions'**
  String get settings_offline_regions_title;

  /// No description provided for @regions_search_hint.
  ///
  /// In en, this message translates to:
  /// **'Search regions'**
  String get regions_search_hint;

  /// No description provided for @regions_dem_toggle_label.
  ///
  /// In en, this message translates to:
  /// **'Download elevation data (DEM)'**
  String get regions_dem_toggle_label;

  /// No description provided for @regions_dem_toggle_caption.
  ///
  /// In en, this message translates to:
  /// **'Adds hillshading; increases download size'**
  String get regions_dem_toggle_caption;

  /// No description provided for @regions_update_available.
  ///
  /// In en, this message translates to:
  /// **'Update available'**
  String get regions_update_available;

  /// No description provided for @regions_update_action.
  ///
  /// In en, this message translates to:
  /// **'Update'**
  String get regions_update_action;

  /// No description provided for @regions_retry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get regions_retry;

  /// No description provided for @regions_not_yet_available.
  ///
  /// In en, this message translates to:
  /// **'Not yet available'**
  String get regions_not_yet_available;

  /// No description provided for @regions_build_failed.
  ///
  /// In en, this message translates to:
  /// **'Build failed'**
  String get regions_build_failed;

  /// No description provided for @regions_delete_confirm_title.
  ///
  /// In en, this message translates to:
  /// **'Delete {name}?'**
  String regions_delete_confirm_title(String name);

  /// No description provided for @regions_delete_confirm_body.
  ///
  /// In en, this message translates to:
  /// **'This removes the downloaded map and elevation data for this region. You\'ll need to download it again to use it offline.'**
  String get regions_delete_confirm_body;

  /// No description provided for @regions_delete_confirm_action.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get regions_delete_confirm_action;

  /// No description provided for @regions_disk_usage_summary.
  ///
  /// In en, this message translates to:
  /// **'{size} used across {count} downloaded region(s)'**
  String regions_disk_usage_summary(String size, num count);

  /// No description provided for @regions_empty_search_title.
  ///
  /// In en, this message translates to:
  /// **'No matching regions'**
  String get regions_empty_search_title;

  /// No description provided for @regions_empty_search_body.
  ///
  /// In en, this message translates to:
  /// **'Try a different search term.'**
  String get regions_empty_search_body;

  /// No description provided for @regions_empty_catalog_title.
  ///
  /// In en, this message translates to:
  /// **'No offline regions available'**
  String get regions_empty_catalog_title;

  /// No description provided for @regions_empty_catalog_body.
  ///
  /// In en, this message translates to:
  /// **'Ask your Wanderer instance administrator to configure downloadable regions.'**
  String get regions_empty_catalog_body;

  /// No description provided for @regions_vector_tile_title.
  ///
  /// In en, this message translates to:
  /// **'Vector'**
  String get regions_vector_tile_title;

  /// No description provided for @regions_dem_tile_title.
  ///
  /// In en, this message translates to:
  /// **'Elevation data'**
  String get regions_dem_tile_title;

  /// No description provided for @regions_download_failed.
  ///
  /// In en, this message translates to:
  /// **'Download failed'**
  String get regions_download_failed;

  /// No description provided for @regions_dem_locked_subtitle.
  ///
  /// In en, this message translates to:
  /// **'Download map data first'**
  String get regions_dem_locked_subtitle;

  /// No description provided for @regions_offline_unavailable_title.
  ///
  /// In en, this message translates to:
  /// **'Can\'t load regions'**
  String get regions_offline_unavailable_title;

  /// No description provided for @regions_offline_unavailable_body.
  ///
  /// In en, this message translates to:
  /// **'Connect to the internet to browse and manage downloadable regions.'**
  String get regions_offline_unavailable_body;

  /// No description provided for @regions_map_geometry_failed.
  ///
  /// In en, this message translates to:
  /// **'Could not load region outline'**
  String get regions_map_geometry_failed;

  /// No description provided for @regions_map_back_label.
  ///
  /// In en, this message translates to:
  /// **'Back to regions'**
  String get regions_map_back_label;

  /// No description provided for @regions_group_expand_label.
  ///
  /// In en, this message translates to:
  /// **'Expand {name}'**
  String regions_group_expand_label(String name);

  /// No description provided for @regions_group_collapse_label.
  ///
  /// In en, this message translates to:
  /// **'Collapse {name}'**
  String regions_group_collapse_label(String name);

  /// Title shown by the shared offline takeover state (map, list, profile) when the backend is unreachable.
  ///
  /// In en, this message translates to:
  /// **'You\'re offline'**
  String get offline_title;

  /// Retry button label on the shared offline takeover state.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get offline_try_again;

  /// Body copy for the map screen's offline takeover.
  ///
  /// In en, this message translates to:
  /// **'Connect to the internet to load the map. Downloaded trails are still available.'**
  String get offline_map_body;

  /// Body copy for the list screen's offline takeover.
  ///
  /// In en, this message translates to:
  /// **'Connect to the internet to load lists.'**
  String get offline_list_body;

  /// Body copy for the profile screen's offline takeover of network-only sections.
  ///
  /// In en, this message translates to:
  /// **'Connect to the internet to load your full profile.'**
  String get offline_profile_body;

  /// No description provided for @offline_settings_banner.
  ///
  /// In en, this message translates to:
  /// **'You\'re offline. Settings are read-only until you reconnect.'**
  String get offline_settings_banner;

  /// No description provided for @offline_action_unavailable.
  ///
  /// In en, this message translates to:
  /// **'You\'re offline — try again once you\'re back online.'**
  String get offline_action_unavailable;

  /// Body copy for the categories and subcategories settings screens' offline state.
  ///
  /// In en, this message translates to:
  /// **'Connect to the internet to manage categories.'**
  String get offline_categories_body;

  /// Body copy shown inside the map screen's draggable trail-results sheet when offline.
  ///
  /// In en, this message translates to:
  /// **'Connect to the internet to search for trails. Downloaded trails are still available.'**
  String get offline_trail_search_body;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) => <String>[
    'cs',
    'de',
    'en',
    'es',
    'eu',
    'fr',
    'hu',
    'it',
    'nl',
    'no',
    'pl',
    'pt',
    'ru',
    'zh',
  ].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'cs':
      return AppLocalizationsCs();
    case 'de':
      return AppLocalizationsDe();
    case 'en':
      return AppLocalizationsEn();
    case 'es':
      return AppLocalizationsEs();
    case 'eu':
      return AppLocalizationsEu();
    case 'fr':
      return AppLocalizationsFr();
    case 'hu':
      return AppLocalizationsHu();
    case 'it':
      return AppLocalizationsIt();
    case 'nl':
      return AppLocalizationsNl();
    case 'no':
      return AppLocalizationsNo();
    case 'pl':
      return AppLocalizationsPl();
    case 'pt':
      return AppLocalizationsPt();
    case 'ru':
      return AppLocalizationsRu();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
