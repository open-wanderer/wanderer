import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_de.dart';
import 'app_localizations_en.dart';

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
    Locale('de'),
    Locale('en'),
  ];

  /// No description provided for @biking.
  ///
  /// In en, this message translates to:
  /// **'Biking'**
  String get biking;

  /// No description provided for @canoeing.
  ///
  /// In en, this message translates to:
  /// **'Canoeing'**
  String get canoeing;

  /// No description provided for @walking.
  ///
  /// In en, this message translates to:
  /// **'Walking'**
  String get walking;

  /// No description provided for @about.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get about;

  /// No description provided for @appearance.
  ///
  /// In en, this message translates to:
  /// **'Appearance'**
  String get appearance;

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

  /// No description provided for @activity.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Activity} other {Activities}}'**
  String activity(num n);

  /// No description provided for @add_bio.
  ///
  /// In en, this message translates to:
  /// **'Add Bio'**
  String get add_bio;

  /// No description provided for @add_entry.
  ///
  /// In en, this message translates to:
  /// **'Add Entry'**
  String get add_entry;

  /// No description provided for @add_to_list.
  ///
  /// In en, this message translates to:
  /// **'Manage lists'**
  String get add_to_list;

  /// No description provided for @add_waypoint.
  ///
  /// In en, this message translates to:
  /// **'Add Waypoint'**
  String get add_waypoint;

  /// No description provided for @added_trail_to.
  ///
  /// In en, this message translates to:
  /// **'Added trail to'**
  String get added_trail_to;

  /// No description provided for @added_trails_to.
  ///
  /// In en, this message translates to:
  /// **'Added trails to'**
  String get added_trails_to;

  /// No description provided for @after.
  ///
  /// In en, this message translates to:
  /// **'After'**
  String get after;

  /// No description provided for @all.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get all;

  /// No description provided for @all_activities.
  ///
  /// In en, this message translates to:
  /// **'All activities'**
  String get all_activities;

  /// No description provided for @allow_auto_geolocate.
  ///
  /// In en, this message translates to:
  /// **'Begin drawing a new trail from your current location'**
  String get allow_auto_geolocate;

  /// No description provided for @alphabetical.
  ///
  /// In en, this message translates to:
  /// **'Alphabetical'**
  String get alphabetical;

  /// No description provided for @already_account.
  ///
  /// In en, this message translates to:
  /// **'Already have an account?'**
  String get already_account;

  /// No description provided for @altitude.
  ///
  /// In en, this message translates to:
  /// **'Altitude'**
  String get altitude;

  /// No description provided for @amenity.
  ///
  /// In en, this message translates to:
  /// **'Amenity'**
  String get amenity;

  /// No description provided for @api_documentation.
  ///
  /// In en, this message translates to:
  /// **'API Documentation'**
  String get api_documentation;

  /// No description provided for @api_tokens.
  ///
  /// In en, this message translates to:
  /// **'API Tokens'**
  String get api_tokens;

  /// No description provided for @api_tokens_hint.
  ///
  /// In en, this message translates to:
  /// **'API Tokens can be used to grant 3rd party applications access to your wanderer account.'**
  String get api_tokens_hint;

  /// No description provided for @apply_user_settings.
  ///
  /// In en, this message translates to:
  /// **'Apply user settings'**
  String get apply_user_settings;

  /// No description provided for @attraction.
  ///
  /// In en, this message translates to:
  /// **'Attraction'**
  String get attraction;

  /// No description provided for @author.
  ///
  /// In en, this message translates to:
  /// **'Author'**
  String get author;

  /// No description provided for @avatar.
  ///
  /// In en, this message translates to:
  /// **'Avatar'**
  String get avatar;

  /// No description provided for @average_speed.
  ///
  /// In en, this message translates to:
  /// **'Avg. Speed'**
  String get average_speed;

  /// No description provided for @avoid_bad_surfaces.
  ///
  /// In en, this message translates to:
  /// **'Avoid Bad Surfaces'**
  String get avoid_bad_surfaces;

  /// No description provided for @back.
  ///
  /// In en, this message translates to:
  /// **'Back'**
  String get back;

  /// No description provided for @back_to_login.
  ///
  /// In en, this message translates to:
  /// **'Back to Login'**
  String get back_to_login;

  /// No description provided for @bakery.
  ///
  /// In en, this message translates to:
  /// **'Bakery'**
  String get bakery;

  /// No description provided for @barrier.
  ///
  /// In en, this message translates to:
  /// **'Barrier'**
  String get barrier;

  /// No description provided for @basic_info.
  ///
  /// In en, this message translates to:
  /// **'Basic Info'**
  String get basic_info;

  /// No description provided for @basque.
  ///
  /// In en, this message translates to:
  /// **'Basque'**
  String get basque;

  /// No description provided for @before.
  ///
  /// In en, this message translates to:
  /// **'Before'**
  String get before;

  /// No description provided for @behavior.
  ///
  /// In en, this message translates to:
  /// **'Behavior'**
  String get behavior;

  /// No description provided for @bicycle_parking.
  ///
  /// In en, this message translates to:
  /// **'Bicycle Parking'**
  String get bicycle_parking;

  /// No description provided for @bicycle_rental.
  ///
  /// In en, this message translates to:
  /// **'Bicycle Rental'**
  String get bicycle_rental;

  /// No description provided for @bicycle_shop.
  ///
  /// In en, this message translates to:
  /// **'Bicycle Shop'**
  String get bicycle_shop;

  /// No description provided for @bike_type.
  ///
  /// In en, this message translates to:
  /// **'Bike Type'**
  String get bike_type;

  /// No description provided for @bus_stop.
  ///
  /// In en, this message translates to:
  /// **'Bus stop'**
  String get bus_stop;

  /// No description provided for @by.
  ///
  /// In en, this message translates to:
  /// **'by'**
  String get by;

  /// No description provided for @campsite.
  ///
  /// In en, this message translates to:
  /// **'Campsite'**
  String get campsite;

  /// No description provided for @can.
  ///
  /// In en, this message translates to:
  /// **'can'**
  String get can;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @car.
  ///
  /// In en, this message translates to:
  /// **'Car'**
  String get car;

  /// No description provided for @car_motorcycle.
  ///
  /// In en, this message translates to:
  /// **'Car/Motorcycle'**
  String get car_motorcycle;

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

  /// No description provided for @change.
  ///
  /// In en, this message translates to:
  /// **'Change'**
  String get change;

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

  /// No description provided for @changelog.
  ///
  /// In en, this message translates to:
  /// **'Changelog'**
  String get changelog;

  /// No description provided for @chinese.
  ///
  /// In en, this message translates to:
  /// **'Chinese (simplified)'**
  String get chinese;

  /// No description provided for @clear_all.
  ///
  /// In en, this message translates to:
  /// **'Clear all'**
  String get clear_all;

  /// No description provided for @climbing.
  ///
  /// In en, this message translates to:
  /// **'Climbing'**
  String get climbing;

  /// No description provided for @close.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get close;

  /// No description provided for @collapse_trail_list.
  ///
  /// In en, this message translates to:
  /// **'Collapse trail list'**
  String get collapse_trail_list;

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

  /// No description provided for @completed_a_trail.
  ///
  /// In en, this message translates to:
  /// **'completed a trail'**
  String get completed_a_trail;

  /// No description provided for @completed_tours.
  ///
  /// In en, this message translates to:
  /// **'Completed tours'**
  String get completed_tours;

  /// No description provided for @completion_status.
  ///
  /// In en, this message translates to:
  /// **'Completion Status'**
  String get completion_status;

  /// No description provided for @confirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get confirm;

  /// No description provided for @confirm_deletion.
  ///
  /// In en, this message translates to:
  /// **'Confirm Deletion'**
  String get confirm_deletion;

  /// No description provided for @confirm_publish.
  ///
  /// In en, this message translates to:
  /// **'Confirm publishing'**
  String get confirm_publish;

  /// No description provided for @confirm_share.
  ///
  /// In en, this message translates to:
  /// **'Confirm share'**
  String get confirm_share;

  /// No description provided for @connect.
  ///
  /// In en, this message translates to:
  /// **'Connect'**
  String get connect;

  /// No description provided for @contribute.
  ///
  /// In en, this message translates to:
  /// **'Contribute'**
  String get contribute;

  /// No description provided for @couldnt_start_navigation.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t start navigation. Check your connection and try again.'**
  String get couldnt_start_navigation;

  /// No description provided for @copy_link.
  ///
  /// In en, this message translates to:
  /// **'Copy Link'**
  String get copy_link;

  /// No description provided for @create_new_list.
  ///
  /// In en, this message translates to:
  /// **'Create new list'**
  String get create_new_list;

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

  /// No description provided for @crop.
  ///
  /// In en, this message translates to:
  /// **'Crop'**
  String get crop;

  /// No description provided for @cross.
  ///
  /// In en, this message translates to:
  /// **'Cross'**
  String get cross;

  /// No description provided for @current_password.
  ///
  /// In en, this message translates to:
  /// **'Current password'**
  String get current_password;

  /// No description provided for @cycling.
  ///
  /// In en, this message translates to:
  /// **'Cycling'**
  String get cycling;

  /// No description provided for @cycling_speed.
  ///
  /// In en, this message translates to:
  /// **'Cycling Speed'**
  String get cycling_speed;

  /// No description provided for @czech.
  ///
  /// In en, this message translates to:
  /// **'Czech'**
  String get czech;

  /// No description provided for @danger_zone.
  ///
  /// In en, this message translates to:
  /// **'Danger zone'**
  String get danger_zone;

  /// No description provided for @date.
  ///
  /// In en, this message translates to:
  /// **'Date'**
  String get date;

  /// No description provided for @default_category.
  ///
  /// In en, this message translates to:
  /// **'Default category'**
  String get default_category;

  /// No description provided for @default_location.
  ///
  /// In en, this message translates to:
  /// **'Default Location'**
  String get default_location;

  /// No description provided for @degrees.
  ///
  /// In en, this message translates to:
  /// **'Degrees'**
  String get degrees;

  /// No description provided for @delete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get delete;

  /// No description provided for @open.
  ///
  /// In en, this message translates to:
  /// **'Open'**
  String get open;

  /// No description provided for @delete_account.
  ///
  /// In en, this message translates to:
  /// **'Delete Account'**
  String get delete_account;

  /// No description provided for @delete_list_confirm.
  ///
  /// In en, this message translates to:
  /// **'Do you really want to delete this list? The trails in the list will still be available.'**
  String get delete_list_confirm;

  /// No description provided for @delete_summit_log_confirm.
  ///
  /// In en, this message translates to:
  /// **'Do you really want to delete this summit log? This action cannot be undone.'**
  String get delete_summit_log_confirm;

  /// No description provided for @delete_trail_confirm.
  ///
  /// In en, this message translates to:
  /// **'Do you really want to delete this trail? This action cannot be undone.'**
  String get delete_trail_confirm;

  /// No description provided for @describe_your_trail.
  ///
  /// In en, this message translates to:
  /// **'Describe your trail'**
  String get describe_your_trail;

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

  /// No description provided for @display.
  ///
  /// In en, this message translates to:
  /// **'Display'**
  String get display;

  /// No description provided for @display_as.
  ///
  /// In en, this message translates to:
  /// **'Display as'**
  String get display_as;

  /// No description provided for @distance.
  ///
  /// In en, this message translates to:
  /// **'Distance'**
  String get distance;

  /// No description provided for @documentation.
  ///
  /// In en, this message translates to:
  /// **'Documentation'**
  String get documentation;

  /// No description provided for @download.
  ///
  /// In en, this message translates to:
  /// **'Download'**
  String get download;

  /// No description provided for @draw_a_route.
  ///
  /// In en, this message translates to:
  /// **'Draw a route'**
  String get draw_a_route;

  /// No description provided for @driving.
  ///
  /// In en, this message translates to:
  /// **'Driving'**
  String get driving;

  /// No description provided for @duplicate.
  ///
  /// In en, this message translates to:
  /// **'Duplicate'**
  String get duplicate;

  /// No description provided for @duration.
  ///
  /// In en, this message translates to:
  /// **'Duration'**
  String get duration;

  /// No description provided for @dutch.
  ///
  /// In en, this message translates to:
  /// **'Dutch'**
  String get dutch;

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

  /// No description provided for @edit_entry.
  ///
  /// In en, this message translates to:
  /// **'Edit Entry'**
  String get edit_entry;

  /// No description provided for @edit_list.
  ///
  /// In en, this message translates to:
  /// **'Edit List'**
  String get edit_list;

  /// No description provided for @edit_route.
  ///
  /// In en, this message translates to:
  /// **'Edit route'**
  String get edit_route;

  /// No description provided for @edit_waypoint.
  ///
  /// In en, this message translates to:
  /// **'Edit Waypoint'**
  String get edit_waypoint;

  /// No description provided for @edited.
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

  /// No description provided for @email_verified.
  ///
  /// In en, this message translates to:
  /// **'Email verified'**
  String get email_verified;

  /// No description provided for @empty_activities.
  ///
  /// In en, this message translates to:
  /// **'{username} has no activity yet'**
  String empty_activities(Object username);

  /// No description provided for @empty_bio.
  ///
  /// In en, this message translates to:
  /// **'{username} has not added a bio yet'**
  String empty_bio(Object username);

  /// No description provided for @empty_feed.
  ///
  /// In en, this message translates to:
  /// **'Your feed is empty'**
  String get empty_feed;

  /// No description provided for @empty_feed_explanation.
  ///
  /// In en, this message translates to:
  /// **'Activities by you or people you follow will appear here'**
  String get empty_feed_explanation;

  /// No description provided for @empty_lists.
  ///
  /// In en, this message translates to:
  /// **'{username} has no public lists'**
  String empty_lists(Object username);

  /// No description provided for @enable_auto_routing.
  ///
  /// In en, this message translates to:
  /// **'Enable auto_routing'**
  String get enable_auto_routing;

  /// No description provided for @english.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get english;

  /// No description provided for @entry.
  ///
  /// In en, this message translates to:
  /// **'Entry'**
  String get entry;

  /// No description provided for @error_copying_trail.
  ///
  /// In en, this message translates to:
  /// **'Error copying trail'**
  String get error_copying_trail;

  /// No description provided for @error_creating_user.
  ///
  /// In en, this message translates to:
  /// **'Error creating user'**
  String get error_creating_user;

  /// No description provided for @error_deleting_token.
  ///
  /// In en, this message translates to:
  /// **'Error deleting token'**
  String get error_deleting_token;

  /// No description provided for @error_disabling_strava_integration.
  ///
  /// In en, this message translates to:
  /// **'Error disabling strava integration'**
  String get error_disabling_strava_integration;

  /// No description provided for @error_during_login.
  ///
  /// In en, this message translates to:
  /// **'Error during login'**
  String get error_during_login;

  /// No description provided for @error_during_password_reset.
  ///
  /// In en, this message translates to:
  /// **'Unable to send password reset email'**
  String get error_during_password_reset;

  /// No description provided for @error_exporting_trail.
  ///
  /// In en, this message translates to:
  /// **'Error exporting trail'**
  String get error_exporting_trail;

  /// No description provided for @error_generating_token.
  ///
  /// In en, this message translates to:
  /// **'Error generating token'**
  String get error_generating_token;

  /// No description provided for @error_liking_trail.
  ///
  /// In en, this message translates to:
  /// **'Error liking trail'**
  String get error_liking_trail;

  /// No description provided for @error_logging_in_to_hammerhead.
  ///
  /// In en, this message translates to:
  /// **'Error logging in to Hammerhead'**
  String get error_logging_in_to_hammerhead;

  /// No description provided for @error_logging_in_to_komoot.
  ///
  /// In en, this message translates to:
  /// **'Error logging in to komoot'**
  String get error_logging_in_to_komoot;

  /// No description provided for @error_posting_comment.
  ///
  /// In en, this message translates to:
  /// **'Error posting comment'**
  String get error_posting_comment;

  /// No description provided for @error_printing_map.
  ///
  /// In en, this message translates to:
  /// **'Error printing map'**
  String get error_printing_map;

  /// No description provided for @error_reading_file.
  ///
  /// In en, this message translates to:
  /// **'Error reading file'**
  String get error_reading_file;

  /// No description provided for @error_saving_list.
  ///
  /// In en, this message translates to:
  /// **'Error saving list'**
  String get error_saving_list;

  /// No description provided for @error_saving_trail.
  ///
  /// In en, this message translates to:
  /// **'Error saving trail'**
  String get error_saving_trail;

  /// No description provided for @error_setting_up_integration.
  ///
  /// In en, this message translates to:
  /// **'Error setting up {provider} integration'**
  String error_setting_up_integration(Object provider);

  /// No description provided for @error_updating_hammerhead_integration.
  ///
  /// In en, this message translates to:
  /// **'Error updating Hammerhead integration'**
  String get error_updating_hammerhead_integration;

  /// No description provided for @error_updating_komoot_integration.
  ///
  /// In en, this message translates to:
  /// **'Error updating komoot integration'**
  String get error_updating_komoot_integration;

  /// No description provided for @error_updating_password.
  ///
  /// In en, this message translates to:
  /// **'Error updating password'**
  String get error_updating_password;

  /// No description provided for @error_updating_strava_integration.
  ///
  /// In en, this message translates to:
  /// **'Error updating Strava integration'**
  String get error_updating_strava_integration;

  /// No description provided for @error_uploading_trail_to_hammerhead.
  ///
  /// In en, this message translates to:
  /// **'Error uploading trail to Hammerhead'**
  String get error_uploading_trail_to_hammerhead;

  /// No description provided for @est_duration.
  ///
  /// In en, this message translates to:
  /// **'Est. duration'**
  String get est_duration;

  /// No description provided for @everyone_with_the_link.
  ///
  /// In en, this message translates to:
  /// **'Everyone with the link'**
  String get everyone_with_the_link;

  /// No description provided for @expand_trail_list.
  ///
  /// In en, this message translates to:
  /// **'Expand trail list'**
  String get expand_trail_list;

  /// No description provided for @expiration.
  ///
  /// In en, this message translates to:
  /// **'Expiration'**
  String get expiration;

  /// No description provided for @expires.
  ///
  /// In en, this message translates to:
  /// **'Expires'**
  String get expires;

  /// No description provided for @explore.
  ///
  /// In en, this message translates to:
  /// **'Explore'**
  String get explore;

  /// No description provided for @explore_some_trails.
  ///
  /// In en, this message translates to:
  /// **'Explore some trails'**
  String get explore_some_trails;

  /// No description provided for @exit_navigation.
  ///
  /// In en, this message translates to:
  /// **'Exit'**
  String get exit_navigation;

  /// No description provided for @export.
  ///
  /// In en, this message translates to:
  /// **'Export'**
  String get export;

  /// No description provided for @export_all_trails.
  ///
  /// In en, this message translates to:
  /// **'Export all trails'**
  String get export_all_trails;

  /// No description provided for @favourite_sport.
  ///
  /// In en, this message translates to:
  /// **'Favourite sport'**
  String get favourite_sport;

  /// No description provided for @features.
  ///
  /// In en, this message translates to:
  /// **'Features'**
  String get features;

  /// No description provided for @ferry.
  ///
  /// In en, this message translates to:
  /// **'Ferry'**
  String get ferry;

  /// No description provided for @file_format.
  ///
  /// In en, this message translates to:
  /// **'File format'**
  String get file_format;

  /// No description provided for @file_too_big.
  ///
  /// In en, this message translates to:
  /// **'File {file} is too big (max. {size})'**
  String file_too_big(Object file, Object size);

  /// No description provided for @filter_categories.
  ///
  /// In en, this message translates to:
  /// **'Filter categories'**
  String get filter_categories;

  /// No description provided for @filter_difficulty.
  ///
  /// In en, this message translates to:
  /// **'Filter difficulty'**
  String get filter_difficulty;

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

  /// No description provided for @finish.
  ///
  /// In en, this message translates to:
  /// **'Finish'**
  String get finish;

  /// No description provided for @fixed_speed.
  ///
  /// In en, this message translates to:
  /// **'Fixed Speed'**
  String get fixed_speed;

  /// No description provided for @focus_map_on.
  ///
  /// In en, this message translates to:
  /// **'Focus map on'**
  String get focus_map_on;

  /// No description provided for @follow.
  ///
  /// In en, this message translates to:
  /// **'Follow'**
  String get follow;

  /// No description provided for @follow_request_pending.
  ///
  /// In en, this message translates to:
  /// **'Request pending'**
  String get follow_request_pending;

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

  /// No description provided for @food.
  ///
  /// In en, this message translates to:
  /// **'Food'**
  String get food;

  /// No description provided for @food_drinks.
  ///
  /// In en, this message translates to:
  /// **'Food & Drinks'**
  String get food_drinks;

  /// No description provided for @forgot_your_password.
  ///
  /// In en, this message translates to:
  /// **'Forgot your password?'**
  String get forgot_your_password;

  /// No description provided for @french.
  ///
  /// In en, this message translates to:
  /// **'French'**
  String get french;

  /// No description provided for @from_file.
  ///
  /// In en, this message translates to:
  /// **'From file'**
  String get from_file;

  /// No description provided for @from_photos.
  ///
  /// In en, this message translates to:
  /// **'From Photos'**
  String get from_photos;

  /// No description provided for @from_url.
  ///
  /// In en, this message translates to:
  /// **'From URL'**
  String get from_url;

  /// No description provided for @garage.
  ///
  /// In en, this message translates to:
  /// **'Garage'**
  String get garage;

  /// No description provided for @gas_station.
  ///
  /// In en, this message translates to:
  /// **'Gas station'**
  String get gas_station;

  /// No description provided for @generate_new_token.
  ///
  /// In en, this message translates to:
  /// **'Generate new token'**
  String get generate_new_token;

  /// No description provided for @german.
  ///
  /// In en, this message translates to:
  /// **'German'**
  String get german;

  /// No description provided for @get_position_from_exif.
  ///
  /// In en, this message translates to:
  /// **'Get coordinates from EXIF data'**
  String get get_position_from_exif;

  /// No description provided for @get_started.
  ///
  /// In en, this message translates to:
  /// **'Get started'**
  String get get_started;

  /// No description provided for @grid.
  ///
  /// In en, this message translates to:
  /// **'Grid'**
  String get grid;

  /// No description provided for @grocery_store.
  ///
  /// In en, this message translates to:
  /// **'Grocery store'**
  String get grocery_store;

  /// No description provided for @hammerhead_integration_after_date_hint.
  ///
  /// In en, this message translates to:
  /// **'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.'**
  String get hammerhead_integration_after_date_hint;

  /// No description provided for @heading.
  ///
  /// In en, this message translates to:
  /// **'Heading'**
  String get heading;

  /// No description provided for @height.
  ///
  /// In en, this message translates to:
  /// **'Height'**
  String get height;

  /// No description provided for @help.
  ///
  /// In en, this message translates to:
  /// **'Help'**
  String get help;

  /// No description provided for @hero_section_0_text.
  ///
  /// In en, this message translates to:
  /// **'Explore exciting trails, save your favorites, and experience the beauty of nature. Find your next adventure!'**
  String get hero_section_0_text;

  /// No description provided for @hero_section_1_heading.
  ///
  /// In en, this message translates to:
  /// **'It seems there are no trails here yet.'**
  String get hero_section_1_heading;

  /// No description provided for @hero_section_1_text.
  ///
  /// In en, this message translates to:
  /// **'Here are some trails you might like. Or you can just go to the full list right now.'**
  String get hero_section_1_text;

  /// No description provided for @hero_section_1_text_alternative.
  ///
  /// In en, this message translates to:
  /// **'Get started by saving your latest adventure.'**
  String get hero_section_1_text_alternative;

  /// No description provided for @hero_section_2_text.
  ///
  /// In en, this message translates to:
  /// **'Did you know? You cannot only save you hiking trails. There are many categories for all your adventures.'**
  String get hero_section_2_text;

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

  /// No description provided for @hotel.
  ///
  /// In en, this message translates to:
  /// **'Hotel'**
  String get hotel;

  /// No description provided for @hungarian.
  ///
  /// In en, this message translates to:
  /// **'Hungarian'**
  String get hungarian;

  /// No description provided for @hut.
  ///
  /// In en, this message translates to:
  /// **'Hut'**
  String get hut;

  /// No description provided for @hybrid.
  ///
  /// In en, this message translates to:
  /// **'Hybrid'**
  String get hybrid;

  /// No description provided for @icon.
  ///
  /// In en, this message translates to:
  /// **'Icon'**
  String get icon;

  /// No description provided for @ignore_trails_before_date.
  ///
  /// In en, this message translates to:
  /// **'Ignore trails before this date'**
  String get ignore_trails_before_date;

  /// No description provided for @imperial.
  ///
  /// In en, this message translates to:
  /// **'Imperial'**
  String get imperial;

  /// No description provided for @import.
  ///
  /// In en, this message translates to:
  /// **'Import'**
  String get import;

  /// No description provided for @import_hint.
  ///
  /// In en, this message translates to:
  /// **'Select or drag GPX, FIT, KML or TCX files here...'**
  String get import_hint;

  /// No description provided for @include_description.
  ///
  /// In en, this message translates to:
  /// **'Include description'**
  String get include_description;

  /// No description provided for @include_waypoints.
  ///
  /// In en, this message translates to:
  /// **'Include waypoints'**
  String get include_waypoints;

  /// No description provided for @integration_description_hammerhead.
  ///
  /// In en, this message translates to:
  /// **'Syncs your Hammerhead tours with wanderer in regular intervals.'**
  String get integration_description_hammerhead;

  /// No description provided for @integration_description_komoot.
  ///
  /// In en, this message translates to:
  /// **'Syncs your komoot tours with wanderer in regular intervals.'**
  String get integration_description_komoot;

  /// No description provided for @integration_description_strava.
  ///
  /// In en, this message translates to:
  /// **'Syncs your Strava routes & activities with wanderer in regular intervals.'**
  String get integration_description_strava;

  /// No description provided for @integration_disabled.
  ///
  /// In en, this message translates to:
  /// **'integration disabled'**
  String get integration_disabled;

  /// No description provided for @integration_enabled.
  ///
  /// In en, this message translates to:
  /// **'integration enabled'**
  String get integration_enabled;

  /// No description provided for @integration_privacy_hint_original.
  ///
  /// In en, this message translates to:
  /// **'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.'**
  String get integration_privacy_hint_original;

  /// No description provided for @integration_privacy_hint_user.
  ///
  /// In en, this message translates to:
  /// **'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.'**
  String get integration_privacy_hint_user;

  /// No description provided for @integrations.
  ///
  /// In en, this message translates to:
  /// **'Integrations'**
  String get integrations;

  /// No description provided for @invalid_date.
  ///
  /// In en, this message translates to:
  /// **'Invalid Date'**
  String get invalid_date;

  /// No description provided for @invalid_username.
  ///
  /// In en, this message translates to:
  /// **'Invalid username'**
  String get invalid_username;

  /// No description provided for @italian.
  ///
  /// In en, this message translates to:
  /// **'Italian'**
  String get italian;

  /// No description provided for @joined.
  ///
  /// In en, this message translates to:
  /// **'Joined'**
  String get joined;

  /// No description provided for @keep_original.
  ///
  /// In en, this message translates to:
  /// **'Keep original'**
  String get keep_original;

  /// No description provided for @keep_private.
  ///
  /// In en, this message translates to:
  /// **'Keep private'**
  String get keep_private;

  /// No description provided for @language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// No description provided for @last_used.
  ///
  /// In en, this message translates to:
  /// **'Last used'**
  String get last_used;

  /// No description provided for @latitude.
  ///
  /// In en, this message translates to:
  /// **'Latitude'**
  String get latitude;

  /// No description provided for @layer.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Layer} other {Layers}}'**
  String layer(num n);

  /// No description provided for @license.
  ///
  /// In en, this message translates to:
  /// **'License'**
  String get license;

  /// No description provided for @like_status.
  ///
  /// In en, this message translates to:
  /// **'Like Status'**
  String get like_status;

  /// No description provided for @liked.
  ///
  /// In en, this message translates to:
  /// **'Liked'**
  String get liked;

  /// No description provided for @likes.
  ///
  /// In en, this message translates to:
  /// **'Likes'**
  String get likes;

  /// No description provided for @limited.
  ///
  /// In en, this message translates to:
  /// **'Limited'**
  String get limited;

  /// No description provided for @link_copied.
  ///
  /// In en, this message translates to:
  /// **'Link copied!'**
  String get link_copied;

  /// No description provided for @linked_lists.
  ///
  /// In en, this message translates to:
  /// **'Linked lists'**
  String get linked_lists;

  /// No description provided for @list.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {List} other {Lists}}'**
  String list(num n);

  /// No description provided for @list_not_shared.
  ///
  /// In en, this message translates to:
  /// **'Not shared with anyone'**
  String get list_not_shared;

  /// No description provided for @list_public_warning.
  ///
  /// In en, this message translates to:
  /// **'All trails in this list will become public.'**
  String get list_public_warning;

  /// No description provided for @list_saved_successfully.
  ///
  /// In en, this message translates to:
  /// **'List saved successfully'**
  String get list_saved_successfully;

  /// No description provided for @list_share_warning.
  ///
  /// In en, this message translates to:
  /// **'Sharing a list automatically shares all trails contained in it.'**
  String get list_share_warning;

  /// No description provided for @list_share_warning_update.
  ///
  /// In en, this message translates to:
  /// **'Added trails will be shared with everyone that has access to this list.'**
  String get list_share_warning_update;

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

  /// No description provided for @login.
  ///
  /// In en, this message translates to:
  /// **'Login'**
  String get login;

  /// No description provided for @login_details.
  ///
  /// In en, this message translates to:
  /// **'Login details'**
  String get login_details;

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

  /// No description provided for @loop.
  ///
  /// In en, this message translates to:
  /// **'Loop'**
  String get loop;

  /// No description provided for @make_one.
  ///
  /// In en, this message translates to:
  /// **'Make one!'**
  String get make_one;

  /// No description provided for @make_thumbnail.
  ///
  /// In en, this message translates to:
  /// **'Make thumbnail'**
  String get make_thumbnail;

  /// No description provided for @map.
  ///
  /// In en, this message translates to:
  /// **'Map'**
  String get map;

  /// No description provided for @map_style.
  ///
  /// In en, this message translates to:
  /// **'Map style'**
  String get map_style;

  /// No description provided for @max_hiking_difficulty.
  ///
  /// In en, this message translates to:
  /// **'Max. Hiking Difficulty'**
  String get max_hiking_difficulty;

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

  /// No description provided for @more.
  ///
  /// In en, this message translates to:
  /// **'More'**
  String get more;

  /// No description provided for @more_route_settings.
  ///
  /// In en, this message translates to:
  /// **'More route settings'**
  String get more_route_settings;

  /// No description provided for @mountain.
  ///
  /// In en, this message translates to:
  /// **'Mountain'**
  String get mountain;

  /// No description provided for @mountain_pass.
  ///
  /// In en, this message translates to:
  /// **'Mountain pass'**
  String get mountain_pass;

  /// No description provided for @must_be_at_least_n_characters_long.
  ///
  /// In en, this message translates to:
  /// **'Must be at least {n} characters long'**
  String must_be_at_least_n_characters_long(Object n);

  /// No description provided for @must_be_at_most_n_characters_long.
  ///
  /// In en, this message translates to:
  /// **'Must be at most {n} characters long'**
  String must_be_at_most_n_characters_long(Object n);

  /// No description provided for @my_account.
  ///
  /// In en, this message translates to:
  /// **'My Account'**
  String get my_account;

  /// No description provided for @my_trails.
  ///
  /// In en, this message translates to:
  /// **'My trails'**
  String get my_trails;

  /// No description provided for @in_distance.
  ///
  /// In en, this message translates to:
  /// **'in {distance}'**
  String in_distance(String distance);

  /// No description provided for @n_days_ago.
  ///
  /// In en, this message translates to:
  /// **'{n} days ago'**
  String n_days_ago(Object n);

  /// No description provided for @n_hours_ago.
  ///
  /// In en, this message translates to:
  /// **'{n} hours ago'**
  String n_hours_ago(Object n);

  /// No description provided for @n_minutes_ago.
  ///
  /// In en, this message translates to:
  /// **'{n} minutes ago'**
  String n_minutes_ago(Object n);

  /// No description provided for @n_months_ago.
  ///
  /// In en, this message translates to:
  /// **'{n} months ago'**
  String n_months_ago(Object n);

  /// No description provided for @n_seconds_ago.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =0 {just now} other {{n} seconds ago}}'**
  String n_seconds_ago(num n);

  /// No description provided for @n_years_ago.
  ///
  /// In en, this message translates to:
  /// **'{n} years ago'**
  String n_years_ago(Object n);

  /// No description provided for @name.
  ///
  /// In en, this message translates to:
  /// **'Name'**
  String get name;

  /// No description provided for @navigate.
  ///
  /// In en, this message translates to:
  /// **'Navigate'**
  String get navigate;

  /// No description provided for @near.
  ///
  /// In en, this message translates to:
  /// **'Near'**
  String get near;

  /// No description provided for @never.
  ///
  /// In en, this message translates to:
  /// **'Never'**
  String get never;

  /// No description provided for @new_list.
  ///
  /// In en, this message translates to:
  /// **'New List'**
  String get new_list;

  /// No description provided for @new_password.
  ///
  /// In en, this message translates to:
  /// **'New password'**
  String get new_password;

  /// No description provided for @new_password_error.
  ///
  /// In en, this message translates to:
  /// **'Error setting new password'**
  String get new_password_error;

  /// No description provided for @new_password_success.
  ///
  /// In en, this message translates to:
  /// **'The new password has been set'**
  String get new_password_success;

  /// No description provided for @new_password_text.
  ///
  /// In en, this message translates to:
  /// **'Set a new password'**
  String get new_password_text;

  /// No description provided for @new_token_generated.
  ///
  /// In en, this message translates to:
  /// **'New API Token generated'**
  String get new_token_generated;

  /// No description provided for @new_trail.
  ///
  /// In en, this message translates to:
  /// **'New Trail'**
  String get new_trail;

  /// No description provided for @no_account.
  ///
  /// In en, this message translates to:
  /// **'Don\'t have an account?'**
  String get no_account;

  /// No description provided for @no_api_tokens.
  ///
  /// In en, this message translates to:
  /// **'You have no API Tokens'**
  String get no_api_tokens;

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
  /// **'No GPS data in image'**
  String get no_gps_data_in_image;

  /// No description provided for @no_grid.
  ///
  /// In en, this message translates to:
  /// **'No Grid'**
  String get no_grid;

  /// No description provided for @no_notifications.
  ///
  /// In en, this message translates to:
  /// **'No notifications'**
  String get no_notifications;

  /// No description provided for @no_photos_here.
  ///
  /// In en, this message translates to:
  /// **'No photos or videos here'**
  String get no_photos_here;

  /// No description provided for @no_preference.
  ///
  /// In en, this message translates to:
  /// **'No preference'**
  String get no_preference;

  /// No description provided for @no_results.
  ///
  /// In en, this message translates to:
  /// **'No results found'**
  String get no_results;

  /// No description provided for @no_routes_added.
  ///
  /// In en, this message translates to:
  /// **'No routes added'**
  String get no_routes_added;

  /// No description provided for @no_waypoints_yet.
  ///
  /// In en, this message translates to:
  /// **'No waypoints yet'**
  String get no_waypoints_yet;

  /// No description provided for @norwegian.
  ///
  /// In en, this message translates to:
  /// **'Norwegian'**
  String get norwegian;

  /// No description provided for @not_a_valid_email_address.
  ///
  /// In en, this message translates to:
  /// **'Not a valid email address'**
  String get not_a_valid_email_address;

  /// No description provided for @not_a_valid_url.
  ///
  /// In en, this message translates to:
  /// **'Not a valid URL'**
  String get not_a_valid_url;

  /// No description provided for @not_completed.
  ///
  /// In en, this message translates to:
  /// **'Not completed'**
  String get not_completed;

  /// No description provided for @notification_comment_mention.
  ///
  /// In en, this message translates to:
  /// **'{user} mentioned you in a comment'**
  String notification_comment_mention(Object user);

  /// No description provided for @notification_list_create.
  ///
  /// In en, this message translates to:
  /// **'{user} created a new list'**
  String notification_list_create(Object user);

  /// No description provided for @notification_list_share.
  ///
  /// In en, this message translates to:
  /// **'{user} shared a list with you'**
  String notification_list_share(Object user);

  /// No description provided for @notification_new_follower.
  ///
  /// In en, this message translates to:
  /// **'You have a new follower'**
  String get notification_new_follower;

  /// No description provided for @notification_summit_log_create.
  ///
  /// In en, this message translates to:
  /// **'{user} created a summit log on your trail \"{trail}\"'**
  String notification_summit_log_create(Object trail, Object user);

  /// No description provided for @notification_summit_log_mention.
  ///
  /// In en, this message translates to:
  /// **'{user} mentioned you in a summit log'**
  String notification_summit_log_mention(Object user);

  /// No description provided for @notification_trail_comment.
  ///
  /// In en, this message translates to:
  /// **'{user} left a comment on your trail \"{trail}\"'**
  String notification_trail_comment(Object trail, Object user);

  /// No description provided for @notification_trail_create.
  ///
  /// In en, this message translates to:
  /// **'{user} created a new trail'**
  String notification_trail_create(Object user);

  /// No description provided for @notification_trail_like.
  ///
  /// In en, this message translates to:
  /// **'{user} liked your trail \"{trail}\"'**
  String notification_trail_like(Object trail, Object user);

  /// No description provided for @notification_trail_mention.
  ///
  /// In en, this message translates to:
  /// **'{user} mentioned you in a trail'**
  String notification_trail_mention(Object user);

  /// No description provided for @notification_trail_share.
  ///
  /// In en, this message translates to:
  /// **'{user} shared a trail with you'**
  String notification_trail_share(Object user);

  /// No description provided for @notifications.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get notifications;

  /// No description provided for @object_share_error.
  ///
  /// In en, this message translates to:
  /// **'A {object} must be public to be shared across instances.'**
  String object_share_error(Object object);

  /// No description provided for @off.
  ///
  /// In en, this message translates to:
  /// **'Off'**
  String get off;

  /// No description provided for @only_me.
  ///
  /// In en, this message translates to:
  /// **'Only me'**
  String get only_me;

  /// No description provided for @open_in_new_tab.
  ///
  /// In en, this message translates to:
  /// **'Open in new tab'**
  String get open_in_new_tab;

  /// No description provided for @or.
  ///
  /// In en, this message translates to:
  /// **'or'**
  String get or;

  /// No description provided for @orientation.
  ///
  /// In en, this message translates to:
  /// **'Orientation'**
  String get orientation;

  /// No description provided for @paper_size.
  ///
  /// In en, this message translates to:
  /// **'Paper size'**
  String get paper_size;

  /// No description provided for @paragraph.
  ///
  /// In en, this message translates to:
  /// **'Paragraph'**
  String get paragraph;

  /// No description provided for @parking.
  ///
  /// In en, this message translates to:
  /// **'Parking'**
  String get parking;

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

  /// No description provided for @password_reset_sent.
  ///
  /// In en, this message translates to:
  /// **'A password reset email has been sent'**
  String get password_reset_sent;

  /// No description provided for @password_reset_text.
  ///
  /// In en, this message translates to:
  /// **'We will send a reset link to your email address.'**
  String get password_reset_text;

  /// No description provided for @password_updated.
  ///
  /// In en, this message translates to:
  /// **'Password updated'**
  String get password_updated;

  /// No description provided for @passwords_must_match.
  ///
  /// In en, this message translates to:
  /// **'Passwords must match'**
  String get passwords_must_match;

  /// No description provided for @photos.
  ///
  /// In en, this message translates to:
  /// **'Photos & Videos'**
  String get photos;

  /// No description provided for @pick_a_trail.
  ///
  /// In en, this message translates to:
  /// **'Pick a trail'**
  String get pick_a_trail;

  /// No description provided for @planned_a_trail.
  ///
  /// In en, this message translates to:
  /// **'planned a trail'**
  String get planned_a_trail;

  /// No description provided for @planned_tours.
  ///
  /// In en, this message translates to:
  /// **'Planned tours'**
  String get planned_tours;

  /// No description provided for @pois.
  ///
  /// In en, this message translates to:
  /// **'POIs'**
  String get pois;

  /// No description provided for @polish.
  ///
  /// In en, this message translates to:
  /// **'Polish'**
  String get polish;

  /// No description provided for @portuguese.
  ///
  /// In en, this message translates to:
  /// **'Portuguese'**
  String get portuguese;

  /// No description provided for @print.
  ///
  /// In en, this message translates to:
  /// **'Print'**
  String get print;

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

  /// No description provided for @public_access.
  ///
  /// In en, this message translates to:
  /// **'Public access'**
  String get public_access;

  /// No description provided for @public_share_everyone.
  ///
  /// In en, this message translates to:
  /// **'Everyone on the internet with the link can see this trail'**
  String get public_share_everyone;

  /// No description provided for @public_share_limited.
  ///
  /// In en, this message translates to:
  /// **'Only people with access can open the link'**
  String get public_share_limited;

  /// No description provided for @public_transport.
  ///
  /// In en, this message translates to:
  /// **'Public transport'**
  String get public_transport;

  /// No description provided for @radius.
  ///
  /// In en, this message translates to:
  /// **'Radius'**
  String get radius;

  /// No description provided for @railway_station.
  ///
  /// In en, this message translates to:
  /// **'Railway station'**
  String get railway_station;

  /// No description provided for @reached_end_of_trail.
  ///
  /// In en, this message translates to:
  /// **'You\'ve reached the end of the trail.'**
  String get reached_end_of_trail;

  /// No description provided for @read_more.
  ///
  /// In en, this message translates to:
  /// **'Read more'**
  String get read_more;

  /// No description provided for @ready_to_join.
  ///
  /// In en, this message translates to:
  /// **'Ready to join'**
  String get ready_to_join;

  /// No description provided for @recalculate_elevation_data.
  ///
  /// In en, this message translates to:
  /// **'Recalculate elevation data'**
  String get recalculate_elevation_data;

  /// No description provided for @recalculating_elevation_data_hint.
  ///
  /// In en, this message translates to:
  /// **'Recalculating elevation data will erase the existing elevation data, if any, and replace it with data from Valhalla.'**
  String get recalculating_elevation_data_hint;

  /// No description provided for @register.
  ///
  /// In en, this message translates to:
  /// **'Register'**
  String get register;

  /// No description provided for @remote_users_cannot_edit.
  ///
  /// In en, this message translates to:
  /// **'Remote users cannot edit'**
  String get remote_users_cannot_edit;

  /// No description provided for @removed_trail_from.
  ///
  /// In en, this message translates to:
  /// **'Removed trail from'**
  String get removed_trail_from;

  /// No description provided for @removed_trails_from.
  ///
  /// In en, this message translates to:
  /// **'Removed trails from'**
  String get removed_trails_from;

  /// No description provided for @required.
  ///
  /// In en, this message translates to:
  /// **'Required'**
  String get required;

  /// No description provided for @reset.
  ///
  /// In en, this message translates to:
  /// **'Reset'**
  String get reset;

  /// No description provided for @reset_password.
  ///
  /// In en, this message translates to:
  /// **'Reset Password'**
  String get reset_password;

  /// No description provided for @resume.
  ///
  /// In en, this message translates to:
  /// **'Resume'**
  String get resume;

  /// No description provided for @reverse_direction.
  ///
  /// In en, this message translates to:
  /// **'Reverse direction'**
  String get reverse_direction;

  /// No description provided for @road.
  ///
  /// In en, this message translates to:
  /// **'Road'**
  String get road;

  /// No description provided for @route.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Route} other {Routes}}'**
  String route(num n);

  /// No description provided for @route_point.
  ///
  /// In en, this message translates to:
  /// **'Route Point'**
  String get route_point;

  /// No description provided for @russian.
  ///
  /// In en, this message translates to:
  /// **'Russian'**
  String get russian;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @save_list.
  ///
  /// In en, this message translates to:
  /// **'Save List'**
  String get save_list;

  /// No description provided for @save_trail.
  ///
  /// In en, this message translates to:
  /// **'Save Trail'**
  String get save_trail;

  /// No description provided for @save_your_trail_first.
  ///
  /// In en, this message translates to:
  /// **'Save your trail first'**
  String get save_your_trail_first;

  /// No description provided for @search.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get search;

  /// No description provided for @search_cities.
  ///
  /// In en, this message translates to:
  /// **'Search cities'**
  String get search_cities;

  /// No description provided for @search_for_trails_places.
  ///
  /// In en, this message translates to:
  /// **'Search for trails, lists, places'**
  String get search_for_trails_places;

  /// No description provided for @search_list.
  ///
  /// In en, this message translates to:
  /// **'Search list'**
  String get search_list;

  /// No description provided for @search_places.
  ///
  /// In en, this message translates to:
  /// **'Search places'**
  String get search_places;

  /// No description provided for @search_trails.
  ///
  /// In en, this message translates to:
  /// **'Search trails'**
  String get search_trails;

  /// No description provided for @select_date.
  ///
  /// In en, this message translates to:
  /// **'Select date'**
  String get select_date;

  /// No description provided for @select_list.
  ///
  /// In en, this message translates to:
  /// **'Select List'**
  String get select_list;

  /// No description provided for @selected.
  ///
  /// In en, this message translates to:
  /// **'selected'**
  String get selected;

  /// No description provided for @send_to.
  ///
  /// In en, this message translates to:
  /// **'Send to...'**
  String get send_to;

  /// No description provided for @set_private.
  ///
  /// In en, this message translates to:
  /// **'Set private'**
  String get set_private;

  /// No description provided for @set_public.
  ///
  /// In en, this message translates to:
  /// **'Set public'**
  String get set_public;

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

  /// No description provided for @settings_notification_list_create.
  ///
  /// In en, this message translates to:
  /// **'A user who you follow has created a list'**
  String get settings_notification_list_create;

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

  /// No description provided for @settings_notification_trail_create.
  ///
  /// In en, this message translates to:
  /// **'A user who you follow has created a trail'**
  String get settings_notification_trail_create;

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

  /// No description provided for @settings_saved.
  ///
  /// In en, this message translates to:
  /// **'Settings saved'**
  String get settings_saved;

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

  /// No description provided for @share_this_list.
  ///
  /// In en, this message translates to:
  /// **'Share this list'**
  String get share_this_list;

  /// No description provided for @share_this_trail.
  ///
  /// In en, this message translates to:
  /// **'Share this trail'**
  String get share_this_trail;

  /// No description provided for @shared.
  ///
  /// In en, this message translates to:
  /// **'Shared'**
  String get shared;

  /// No description provided for @shared_by.
  ///
  /// In en, this message translates to:
  /// **'Shared by'**
  String get shared_by;

  /// No description provided for @shared_with.
  ///
  /// In en, this message translates to:
  /// **'Shared with'**
  String get shared_with;

  /// No description provided for @shelter.
  ///
  /// In en, this message translates to:
  /// **'Shelter'**
  String get shelter;

  /// No description provided for @shortest.
  ///
  /// In en, this message translates to:
  /// **'shortest'**
  String get shortest;

  /// No description provided for @show_in_overview.
  ///
  /// In en, this message translates to:
  /// **'Show in overview'**
  String get show_in_overview;

  /// No description provided for @show_less.
  ///
  /// In en, this message translates to:
  /// **'Show less'**
  String get show_less;

  /// No description provided for @show_on_map.
  ///
  /// In en, this message translates to:
  /// **'Show on map'**
  String get show_on_map;

  /// No description provided for @shower.
  ///
  /// In en, this message translates to:
  /// **'Shower'**
  String get shower;

  /// No description provided for @skiing.
  ///
  /// In en, this message translates to:
  /// **'Skiing'**
  String get skiing;

  /// No description provided for @slogan.
  ///
  /// In en, this message translates to:
  /// **'Save your adventures!'**
  String get slogan;

  /// No description provided for @slope.
  ///
  /// In en, this message translates to:
  /// **'Slope'**
  String get slope;

  /// No description provided for @someone.
  ///
  /// In en, this message translates to:
  /// **'Someone'**
  String get someone;

  /// No description provided for @sort.
  ///
  /// In en, this message translates to:
  /// **'Sort'**
  String get sort;

  /// No description provided for @spanish.
  ///
  /// In en, this message translates to:
  /// **'Spanish'**
  String get spanish;

  /// No description provided for @speed.
  ///
  /// In en, this message translates to:
  /// **'Speed'**
  String get speed;

  /// No description provided for @start.
  ///
  /// In en, this message translates to:
  /// **'Start'**
  String get start;

  /// No description provided for @statistics.
  ///
  /// In en, this message translates to:
  /// **'Statistics'**
  String get statistics;

  /// No description provided for @stop_drawing.
  ///
  /// In en, this message translates to:
  /// **'Stop drawing'**
  String get stop_drawing;

  /// No description provided for @stop_editing.
  ///
  /// In en, this message translates to:
  /// **'Stop editing'**
  String get stop_editing;

  /// No description provided for @strava_integration_after_date_hint.
  ///
  /// In en, this message translates to:
  /// **'If your account has a large amount of acitivities you may run into Strava\'s API rate limit preventing you from syncing all activities at once. To mitigate this issue you can set an start date below so that only activities that were recorded after this date are synced.'**
  String get strava_integration_after_date_hint;

  /// No description provided for @subway_stop.
  ///
  /// In en, this message translates to:
  /// **'Subway entrance'**
  String get subway_stop;

  /// No description provided for @summit.
  ///
  /// In en, this message translates to:
  /// **'Summit'**
  String get summit;

  /// No description provided for @summit_book.
  ///
  /// In en, this message translates to:
  /// **'Summit Book'**
  String get summit_book;

  /// No description provided for @table.
  ///
  /// In en, this message translates to:
  /// **'Table'**
  String get table;

  /// No description provided for @tags.
  ///
  /// In en, this message translates to:
  /// **'Tags'**
  String get tags;

  /// No description provided for @text.
  ///
  /// In en, this message translates to:
  /// **'Text'**
  String get text;

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

  /// No description provided for @tilesets.
  ///
  /// In en, this message translates to:
  /// **'Custom tilesets'**
  String get tilesets;

  /// No description provided for @time.
  ///
  /// In en, this message translates to:
  /// **'Time'**
  String get time;

  /// No description provided for @toilets.
  ///
  /// In en, this message translates to:
  /// **'Toilets'**
  String get toilets;

  /// No description provided for @top_speed.
  ///
  /// In en, this message translates to:
  /// **'Top Speed'**
  String get top_speed;

  /// No description provided for @tourism.
  ///
  /// In en, this message translates to:
  /// **'Tourism'**
  String get tourism;

  /// No description provided for @trail.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Trail} other {Trails}}'**
  String trail(num n);

  /// No description provided for @trail_copied_successfully.
  ///
  /// In en, this message translates to:
  /// **'trail copied successfully'**
  String get trail_copied_successfully;

  /// No description provided for @trail_has_no_gpx.
  ///
  /// In en, this message translates to:
  /// **'This trail has no GPX data.'**
  String get trail_has_no_gpx;

  /// No description provided for @trail_not_in_list.
  ///
  /// In en, this message translates to:
  /// **'Trail is not in any list'**
  String get trail_not_in_list;

  /// No description provided for @trail_not_shared.
  ///
  /// In en, this message translates to:
  /// **'Not shared with anyone'**
  String get trail_not_shared;

  /// No description provided for @trail_saved_successfully.
  ///
  /// In en, this message translates to:
  /// **'Trail saved successfully'**
  String get trail_saved_successfully;

  /// No description provided for @trails_for_you.
  ///
  /// In en, this message translates to:
  /// **'Trails for you'**
  String get trails_for_you;

  /// No description provided for @tram_stop.
  ///
  /// In en, this message translates to:
  /// **'Tram stop'**
  String get tram_stop;

  /// No description provided for @unchanged.
  ///
  /// In en, this message translates to:
  /// **'unchanged'**
  String get unchanged;

  /// No description provided for @units.
  ///
  /// In en, this message translates to:
  /// **'Units'**
  String get units;

  /// No description provided for @unlink.
  ///
  /// In en, this message translates to:
  /// **'Unlink'**
  String get unlink;

  /// No description provided for @upload_file.
  ///
  /// In en, this message translates to:
  /// **'Upload file'**
  String get upload_file;

  /// No description provided for @upload_gpx.
  ///
  /// In en, this message translates to:
  /// **'Upload GPX'**
  String get upload_gpx;

  /// No description provided for @upload_new_file.
  ///
  /// In en, this message translates to:
  /// **'Upload new file'**
  String get upload_new_file;

  /// No description provided for @uploaded.
  ///
  /// In en, this message translates to:
  /// **'uploaded'**
  String get uploaded;

  /// No description provided for @uploaded_trail_to_hammerhead.
  ///
  /// In en, this message translates to:
  /// **'Successfully uploaded trail to Hammerhead'**
  String get uploaded_trail_to_hammerhead;

  /// No description provided for @use_hills.
  ///
  /// In en, this message translates to:
  /// **'Use hills'**
  String get use_hills;

  /// No description provided for @use_roads.
  ///
  /// In en, this message translates to:
  /// **'Use Roads'**
  String get use_roads;

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

  /// No description provided for @view.
  ///
  /// In en, this message translates to:
  /// **'View'**
  String get view;

  /// No description provided for @viewpoint.
  ///
  /// In en, this message translates to:
  /// **'Viewpoint'**
  String get viewpoint;

  /// No description provided for @visibilty.
  ///
  /// In en, this message translates to:
  /// **'Visibility'**
  String get visibilty;

  /// No description provided for @visibilty_status.
  ///
  /// In en, this message translates to:
  /// **'Visibility status'**
  String get visibilty_status;

  /// No description provided for @walking_speed.
  ///
  /// In en, this message translates to:
  /// **'Walking speed'**
  String get walking_speed;

  /// No description provided for @water.
  ///
  /// In en, this message translates to:
  /// **'Water'**
  String get water;

  /// No description provided for @waypoints.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, =1 {Waypoint} other {Waypoints}}'**
  String waypoints(num n);

  /// No description provided for @welcome_to.
  ///
  /// In en, this message translates to:
  /// **'Welcome to'**
  String get welcome_to;

  /// No description provided for @width.
  ///
  /// In en, this message translates to:
  /// **'Width'**
  String get width;

  /// No description provided for @wrong_username_or_password.
  ///
  /// In en, this message translates to:
  /// **'Wrong username or password'**
  String get wrong_username_or_password;

  /// No description provided for @you_can.
  ///
  /// In en, this message translates to:
  /// **'You can'**
  String get you_can;

  /// No description provided for @you_have_arrived.
  ///
  /// In en, this message translates to:
  /// **'You\'ve arrived'**
  String get you_have_arrived;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['de', 'en'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'de':
      return AppLocalizationsDe();
    case 'en':
      return AppLocalizationsEn();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
