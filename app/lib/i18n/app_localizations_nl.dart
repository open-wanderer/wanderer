// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Dutch Flemish (`nl`).
class AppLocalizationsNl extends AppLocalizations {
  AppLocalizationsNl([String locale = 'nl']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'Over';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Je staat op het punt je account te verwijderen. Al je routes worden hierdoor eveneens verwijderd. Wil je doorgaan?';

  @override
  String get account_privacy => 'Account privacy';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Activiteiten',
      one: 'Activiteit',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Voeg Bio toe';

  @override
  String get add_entry => 'Item toevoegen';

  @override
  String get add_to_list => 'Toevoegen aan lijst';

  @override
  String get add_waypoint => 'Routepunt toevoegen';

  @override
  String get added_trail_to => 'Route toegevoegd aan';

  @override
  String get added_trails_to => 'Route toegevoegd aan';

  @override
  String get after => 'Na';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Alle activiteiten';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Alfabetisch';

  @override
  String get already_account => 'Heb je al een account?';

  @override
  String get altitude => 'Hoogte';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'API-documentatie';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Attractie';

  @override
  String get author => 'Auteur';

  @override
  String get avatar => 'Profielfoto';

  @override
  String get average_speed => 'Gem. Snelheid';

  @override
  String get avoid_bad_surfaces => 'Vermijd slechte ondergrond';

  @override
  String get back => 'Terug';

  @override
  String get back_to_login => 'Terug naar login';

  @override
  String get bakery => 'Bakkerij';

  @override
  String get barrier => 'Barrière';

  @override
  String get basic_info => 'Algemene informatie';

  @override
  String get basque => 'Baskisch';

  @override
  String get before => 'Voor';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Fietsenstalling';

  @override
  String get bicycle_rental => 'Fietsverhuur';

  @override
  String get bicycle_shop => 'Fietsenwinkel';

  @override
  String get bike_type => 'Fiets Type';

  @override
  String get bus_stop => 'Bushalte';

  @override
  String get by => 'door';

  @override
  String get campsite => 'Kampeerplek';

  @override
  String get can => 'kan';

  @override
  String get cancel => 'Annuleren';

  @override
  String get car => 'Wagen';

  @override
  String get car_motorcycle => 'Auto/Motorfiets';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Kaarten',
      one: 'Kaart',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Categorieën';

  @override
  String get category => 'Categorie';

  @override
  String get change => 'Wijzigen';

  @override
  String get change_email => 'E-mail wijzigen';

  @override
  String get change_password => 'Wachtwoord wijzigen';

  @override
  String get changelog => 'Wijzigingslog';

  @override
  String get chinese => 'Chinees (vereenvoudigd)';

  @override
  String get clear_all => 'Wis alles';

  @override
  String get climbing => 'Klimmen';

  @override
  String get close => 'Sluiten';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Opmerkingen',
      one: 'Opmerking',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Voltooid';

  @override
  String get completed_a_trail => 'heeft een route voltooid';

  @override
  String get completed_tours => 'Afgeronde tochten';

  @override
  String get completion_status => 'Voltooiingsstatus';

  @override
  String get confirm => 'Bevestig';

  @override
  String get confirm_deletion => 'Bevestig verwijderen';

  @override
  String get confirm_publish => 'Bevestig publicatie';

  @override
  String get confirm_share => 'Bevestig delen';

  @override
  String get connect => 'Verbind';

  @override
  String get contribute => 'Bijdragen';

  @override
  String get couldnt_start_navigation =>
      'Couldn\'t start navigation. Check your connection and try again.';

  @override
  String get location_services_disabled =>
      'Location services are disabled. Please enable GPS to use navigation.';

  @override
  String get location_permission_denied =>
      'Location permission is required for navigation.';

  @override
  String get location_permission_permanently_denied =>
      'Location permission is permanently denied. Please enable it in Settings.';

  @override
  String get copy_link => 'Kopieer Link';

  @override
  String get create_new_list => 'Nieuwe lijst';

  @override
  String get create_waypoint => 'Nieuw routepunt';

  @override
  String get creation_date => 'Aanmaakdatum';

  @override
  String get crop => 'Bijsnijden';

  @override
  String get cross => 'Oversteken';

  @override
  String get current_password => 'Huidig wachtwoord';

  @override
  String get cycling => 'Fietsen';

  @override
  String get cycling_speed => 'Fietssnelheid';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Gevarenzone';

  @override
  String get date => 'Datum';

  @override
  String get default_category => 'Standaardcategorie';

  @override
  String get default_location => 'Standaardlocatie';

  @override
  String get degrees => 'Graden';

  @override
  String get delete => 'Verwijderen';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Account verwijderen';

  @override
  String get delete_list_confirm =>
      'Weet je zeker dat je deze lijst wilt verwijderen? De routes worden bewaard.';

  @override
  String get delete_summit_log_confirm =>
      'Weet je zeker dat je dit topboek wilt verwijderen? Deze actie is onomkeerbaar.';

  @override
  String get delete_trail_confirm =>
      'Weet je zeker dat je deze route wilt verwijderen? Deze actie is onomkeerbaar.';

  @override
  String get describe_your_trail => 'Omschrijf je wandelroute';

  @override
  String get description => 'Omschrijving';

  @override
  String get difficult => 'Moeilijk';

  @override
  String get difficulty => 'Moeilijkheidsgraad';

  @override
  String get directions => 'Routebeschrijving';

  @override
  String get display => 'Tonen';

  @override
  String get display_as => 'Tonen als';

  @override
  String get distance => 'Afstand';

  @override
  String get documentation => 'Documentatie';

  @override
  String get download => 'Download';

  @override
  String get draw_a_route => 'Teken een route';

  @override
  String get driving => 'Rijden';

  @override
  String get duplicate => 'Dupliceer';

  @override
  String get duration => 'Duur';

  @override
  String get dutch => 'Nederlands';

  @override
  String get easy => 'Makkelijk';

  @override
  String get edit => 'Bewerk';

  @override
  String get edit_entry => 'Bewerk Item';

  @override
  String get edit_list => 'Bewerk Lijst';

  @override
  String get edit_route => 'Bewerk route';

  @override
  String get edit_waypoint => 'Bewerk Routepunt';

  @override
  String get edited => 'bewerkt';

  @override
  String get elevation_gain => 'Hoogteverschil';

  @override
  String get elevation_loss => 'Hoogteverlies';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'E-mail';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Email bijgewerkt';

  @override
  String get email_verified => 'Email geverifieerd';

  @override
  String empty_activities(Object username) {
    return '$username heeft nog geen activiteit';
  }

  @override
  String empty_bio(Object username) {
    return '$username heeft nog geen bio toegevoegd';
  }

  @override
  String get empty_feed => 'Jouw feed is leeg';

  @override
  String get empty_feed_explanation =>
      'Activiteiten van jou of mensen die jij volgt verschijnen hier';

  @override
  String empty_lists(Object username) {
    return '$username heeft geen publieke lijsten';
  }

  @override
  String get enable_auto_routing => 'Automatische routering inschakelen';

  @override
  String get english => 'Engels';

  @override
  String get entry => 'Item';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Fout bij aanmaken gebruiker';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Fout bij het uitschakelen van Strava-integratie';

  @override
  String get error_during_login => 'Het inloggen is mislukt';

  @override
  String get error_during_password_reset =>
      'Kan geen e-mail voor wachtwoordherstel verzenden';

  @override
  String get error_exporting_trail => 'Fout bij exporteren van parcours';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Fout bij het \"leuk vinden\" van route';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Fout tijdens inloggen in Komoot';

  @override
  String get error_posting_comment => 'Fout bij het plaatsen van een reactie';

  @override
  String get error_printing_map => 'Fout bij afdrukken van kaart';

  @override
  String get error_reading_file => 'Fout bij inlezen bestand';

  @override
  String get error_saving_list => 'Fout bij bewaren van lijst';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Fout bij bewaren van route';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Fout bij opzetten $provider integratie';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Fout bij bijwerken van wachtwoord';

  @override
  String get error_updating_strava_integration =>
      'Fout bij bijwerken van Komoot integratie';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Geschatte duur';

  @override
  String get everyone_with_the_link => 'Iedereen met de link';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Verkennen';

  @override
  String get explore_some_trails => 'Verken enkele routes';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Exporteer';

  @override
  String get export_all_trails => 'Exporteer alle routes';

  @override
  String get favourite_sport => 'Favoriete sport';

  @override
  String get features => 'Kenmerken';

  @override
  String get ferry => 'Veerboot';

  @override
  String get file_format => 'Bestandsformaat';

  @override
  String file_too_big(Object file, Object size) {
    return 'Bestand $file is te groot (max. $size)';
  }

  @override
  String get filter_categories => 'Filter categorieën';

  @override
  String get filter_difficulty => 'Filter moeilijkheid';

  @override
  String get filter_tags => 'Filter tags';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Finish';

  @override
  String get fixed_speed => 'Vaste snelheid';

  @override
  String get focus_map_on => 'Focus kaart op';

  @override
  String get follow => 'Volg';

  @override
  String get follow_request_pending => 'Verzoek is in behandeling';

  @override
  String get followers => 'Volgers';

  @override
  String get following => 'Volgend';

  @override
  String get food => 'Eten';

  @override
  String get food_drinks => 'Eten & Drinken';

  @override
  String get forgot_your_password => 'Wachtwoord vergeten?';

  @override
  String get french => 'Frans';

  @override
  String get from_file => 'Van bestand';

  @override
  String get from_photos => 'Van Foto\'s';

  @override
  String get from_url => 'Van URL';

  @override
  String get garage => 'Garage';

  @override
  String get gas_station => 'Benzinestation';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Duits';

  @override
  String get get_position_from_exif => 'Coördinaten ophalen uit EXIF-gegevens';

  @override
  String get get_started => 'Aan de slag';

  @override
  String get grid => 'Rooster';

  @override
  String get grocery_store => 'Kruidenier';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Titel';

  @override
  String get height => 'Hoogte';

  @override
  String get help => 'Help';

  @override
  String get hero_section_0_text =>
      'Explore exciting trails, save your favorites, and experience the beauty of nature. Find your next adventure!';

  @override
  String get hero_section_1_heading => 'It seems there are no trails here yet.';

  @override
  String get hero_section_1_text =>
      'Here are some trails you might like. Or you can just go to the full list right now.';

  @override
  String get hero_section_1_text_alternative =>
      'Get started by saving your latest adventure.';

  @override
  String get hero_section_2_text =>
      'Did you know? You cannot only save you hiking trails. There are many categories for all your adventures.';

  @override
  String get hiking => 'Hiking';

  @override
  String get home => 'Home';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Hongaars';

  @override
  String get hut => 'Hut';

  @override
  String get hybrid => 'Hybride';

  @override
  String get icon => 'Pictogram';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Imperiaal';

  @override
  String get import => 'Importeer';

  @override
  String get import_hint =>
      'Selecteer of sleep GPX-, FIT-, KML- of TCX-bestanden hierheen...';

  @override
  String get include_description => 'Inclusief beschrijving';

  @override
  String get include_waypoints => 'Waypoints toevoegen';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Synchroniseert je Komoot-tochten met Wanderer op regelmatige tijdstippen.';

  @override
  String get integration_description_strava =>
      'Synchroniseert je Strava-routes en -activiteiten met Wanderer op regelmatige tijdstippen.';

  @override
  String get integration_disabled => 'Integratie uitgeschakeld';

  @override
  String get integration_enabled => 'Integratie ingeschakeld';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integraties';

  @override
  String get invalid_date => 'Ongeldige datum';

  @override
  String get invalid_username => 'Ongeldige gebruikersnaam';

  @override
  String get italian => 'Italiaans';

  @override
  String get joined => 'Aangesloten';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Taal';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Breedtegraad';

  @override
  String layer(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Layers',
      one: 'Layer',
    );
    return '$_temp0';
  }

  @override
  String get license => 'Licentie';

  @override
  String get like_status => 'Like status';

  @override
  String get liked => 'Leuk gevonden';

  @override
  String get likes => 'Vind-ik-leuks';

  @override
  String get limited => 'Gelimiteerd';

  @override
  String get link_copied => 'Link gekopieerd';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Lijsten',
      one: 'Lijst',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Niet gedeeld met iemand';

  @override
  String get list_public_warning => 'Alle routes in deze lijst worden publiek';

  @override
  String get list_saved_successfully => 'Lijst succesvol bewaard';

  @override
  String get list_share_warning =>
      'Als u een lijst deelt, worden automatisch ook alle routes die erin voorkomen, gedeeld.';

  @override
  String get list_share_warning_update =>
      'Toegevoegde routes worden gedeeld met iedereen die toegang heeft tot deze lijst.';

  @override
  String get location => 'Locatie';

  @override
  String get locations => 'Locations';

  @override
  String get login => 'Inloggen';

  @override
  String get login_details => 'Login details';

  @override
  String get logout => 'Uitloggen';

  @override
  String get longitude => 'Lengtegraad';

  @override
  String get loop => 'Lus';

  @override
  String get make_one => 'Maak er een aan!';

  @override
  String get make_thumbnail => 'Miniatuur maken';

  @override
  String get map => 'Kaart';

  @override
  String get map_style => 'Kaartstijl';

  @override
  String get max_hiking_difficulty => 'Max. Hiking Moeilijkheid';

  @override
  String get metric => 'Metrisch';

  @override
  String get moderate => 'Gemiddeld';

  @override
  String get more => 'Meer';

  @override
  String get more_route_settings => 'Meer route instellingen';

  @override
  String get mountain => 'Berg';

  @override
  String get mountain_pass => 'Berg pas';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Minimaal $n tekens';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Mag maximaal $n tekens lang zijn.';
  }

  @override
  String get my_account => 'Mijn Account';

  @override
  String get my_trails => 'Mijn paden';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n dagen geleden';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n uuren geleden';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n minuten geleden';
  }

  @override
  String n_months_ago(Object n) {
    return '$n maanden geleden';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n seconden geleden',
      zero: 'zojuist',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n jaren geleden';
  }

  @override
  String get name => 'Naam';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Nabij';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Nieuwe lijst';

  @override
  String get new_password => 'Nieuw wachtwoord';

  @override
  String get new_password_error =>
      'Fout bij het instellen van een nieuw wachtwoord';

  @override
  String get new_password_success => 'Het nieuwe wachtwoord is ingesteld.';

  @override
  String get new_password_text => 'Stel nieuw wachtwoord in';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Nieuwe Route';

  @override
  String get no_account => 'Heb je nog geen account?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Tot nu toe geen opmerkingen';

  @override
  String get no_data => 'Geen data';

  @override
  String get no_description_for_now => 'Voorlopig geen beschrijving';

  @override
  String get no_gps_data_in_image => 'Geen GPS data in afbeelding';

  @override
  String get no_grid => 'Geen raster';

  @override
  String get no_notifications => 'Geen meldingen';

  @override
  String get no_photos_here => 'No photos here';

  @override
  String get no_preference => 'Geen voorkeur';

  @override
  String get no_results => 'Er zijn geen zoekresultaten';

  @override
  String get no_routes_added => 'Geen ';

  @override
  String get no_waypoints_yet => 'Nog geen routepunten';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Het e-mailadres is ongeldig';

  @override
  String get not_a_valid_url => 'Geen geldige URL';

  @override
  String get not_completed => 'Niet voltooid';

  @override
  String notification_comment_mention(Object user) {
    return '$user noemde je in een reactie';
  }

  @override
  String notification_list_create(Object user) {
    return '$user heeft een nieuwe lijst gecreëerd';
  }

  @override
  String notification_list_share(Object user) {
    return '$user heeft een lijst gedeeld met jou';
  }

  @override
  String get notification_new_follower => 'Je hebt een nieuwe volger';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user heeft een topboek aangemaakt voor je route \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user vermeldde je in een topboek';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user heeft een reactie achtergelaten op je wandelroute \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user creëerde en nieuwe wandelroute';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user vindt je route \"$trail\" leuk';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user heeft je in een route genoemd';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user deelde een wandelroute met jou';
  }

  @override
  String get notifications => 'Meldingen';

  @override
  String object_share_error(Object object) {
    return 'Een $object moet publiek zijn om tussen instanties te delen.';
  }

  @override
  String get off => 'Uit';

  @override
  String get only_me => 'Alleen ik';

  @override
  String get open_in_new_tab => 'Open in een nieuwe tabblad';

  @override
  String get or => 'of';

  @override
  String get orientation => 'Oriëntatie';

  @override
  String get paper_size => 'Papierformaat';

  @override
  String get paragraph => 'Alinea';

  @override
  String get parking => 'Parking';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Wachtwoord';

  @override
  String get password_confirm => 'Bevestig wachtwoord';

  @override
  String get password_reset_sent =>
      'Er is een e-mail verzonden om uw wachtwoord opnieuw in te stellen';

  @override
  String get password_reset_text =>
      'Wij sturen een reset link naar uw e-mailadres.';

  @override
  String get password_updated => 'Wachtwoord bijgewerkt';

  @override
  String get passwords_must_match => 'Wachtwoorden moeten overeenkomen';

  @override
  String get photos => 'Foto\'s';

  @override
  String get pick_a_trail => 'Kies een Route';

  @override
  String get planned_a_trail => 'een route gepland';

  @override
  String get planned_tours => 'Geplande tochten';

  @override
  String get pois => 'POIs';

  @override
  String get polish => 'Pools';

  @override
  String get portuguese => 'Portugees';

  @override
  String get print => 'Afdrukken';

  @override
  String get privacy => 'Privacy';

  @override
  String get private => 'Privaat';

  @override
  String get profile => 'Profiel';

  @override
  String get public => 'Openbaar';

  @override
  String get public_access => 'Publieke toegang';

  @override
  String get public_share_everyone =>
      'Iedereen op het internet met de link kan dit pad zien';

  @override
  String get public_share_limited =>
      'Alleen mensen met toegang kunnen de link openen';

  @override
  String get public_transport => 'Openbaar vervoer';

  @override
  String get radius => 'Straal';

  @override
  String get railway_station => 'Treinstation';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Lees meer';

  @override
  String get ready_to_join => 'Klaar om aan te sluiten';

  @override
  String get recalculate_elevation_data => 'Herbereken hoogte gegevens';

  @override
  String get recalculating_elevation_data_hint =>
      'Herberekenen van hoogtegegevens zal de bestaande hoogtegegevens wissen, indien aanwezig, en vervangen door gegevens van Valhalla.';

  @override
  String get register => 'Registreren';

  @override
  String get remote_users_cannot_edit =>
      'Externe gebruikers kunnen niet bewerken';

  @override
  String get removed_trail_from => 'Route verwijderd van';

  @override
  String get removed_trails_from => 'Routes verwijderd van';

  @override
  String get required => 'Verplicht';

  @override
  String get reset => 'Herstellen';

  @override
  String get reset_password => 'Wachtwoord opnieuw instellen';

  @override
  String get resume => 'Resume';

  @override
  String get reverse_direction => 'Omgekeerde richting';

  @override
  String get road => 'Weg';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Tochten',
      one: 'Tocht',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Routepunt';

  @override
  String get russian => 'Russisch';

  @override
  String get save => 'Bewaren';

  @override
  String get save_list => 'Bewaar lijst';

  @override
  String get save_trail => 'Bewaar Route';

  @override
  String get save_your_trail_first => 'Bewaar je route eerst';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Zoeken naar steden';

  @override
  String get search_for_trails_places =>
      'Zoek naar routes, lijsten en locaties';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Zoek plaatsen';

  @override
  String get search_trails => 'Zoek routes';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Kies een lijst';

  @override
  String get selected => 'geselecteerd';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Instellingen';

  @override
  String get settings_notification_comment_mention =>
      'Iemand vermeldt je in een reactie';

  @override
  String get settings_notification_list_create =>
      'Een gebruiker die je volgt, heeft een lijst gecreëerd';

  @override
  String get settings_notification_list_share =>
      'Iemand heeft een lijst met je gedeeld';

  @override
  String get settings_notification_new_follower => 'Je hebt een nieuwe volger';

  @override
  String get settings_notification_summit_log_create =>
      'Iemand heeft een topboek van je trail gemaakt';

  @override
  String get settings_notification_summit_log_mention =>
      'Iemand vermeldde je in een topboek';

  @override
  String get settings_notification_trail_comment =>
      'Iemand heeft een opmerking geplaatst bij je route';

  @override
  String get settings_notification_trail_create =>
      'Een gebruiker die je volgt, heeft een route gecreëerd';

  @override
  String get settings_notification_trail_like => 'Iemand vindt je route leuk';

  @override
  String get settings_notification_trail_mention =>
      'Iemand heeft je in een route genoemd';

  @override
  String get settings_notification_trail_share =>
      'Iemand heeft een route met je gedeeld';

  @override
  String get settings_privacy_account_private =>
      'Alleen jij kunt je profiel zien. Je verschijnt niet in de zoekresultaten. Andere gebruikers kunnen je niet volgen of routes met je delen. Je kunt nog steeds routes of lijsten publiceren.';

  @override
  String get settings_privacy_account_public =>
      'Iedereen kan je profiel zien. Je verschijnt in de zoekresultaten. Andere gebruikers kunnen je volgen en ervaringen met je delen.';

  @override
  String get settings_privacy_lists_private =>
      'Je lijsten zijn standaard privé. Niemand behalve jij kan ze zien. Je kunt deze instelling op elk moment voor individuele lijsten wijzigen.';

  @override
  String get settings_privacy_lists_public =>
      'Je lijsten zijn standaard openbaar. Iedereen kan ze zien. Je kunt deze instelling op elk moment voor individuele lijsten wijzigen.';

  @override
  String get settings_privacy_trails_private =>
      'Je routes zijn standaard privé. Niemand behalve jij kan ze zien. Je kunt deze instelling op elk moment voor individuele routes wijzigen.';

  @override
  String get settings_privacy_trails_public =>
      'Je routes zijn standaard openbaar. Iedereen kan ze zien. Je kunt deze instelling op elk moment voor individuele routes wijzigen.';

  @override
  String get settings_saved => 'Instellingen bewaard';

  @override
  String get share => 'Deel';

  @override
  String get share_profile => 'Deel profiel';

  @override
  String get share_this_list => 'Deel deze lijst';

  @override
  String get share_this_trail => 'Deel deze route';

  @override
  String get shared => 'Gedeeld';

  @override
  String get shared_by => 'Gedeeld door';

  @override
  String get shared_with => 'Gedeeld met';

  @override
  String get shelter => 'Schuilplaats';

  @override
  String get shortest => 'kortste';

  @override
  String get show_in_overview => 'Tonen op overzicht';

  @override
  String get show_less => 'Toon minder';

  @override
  String get show_on_map => 'Tonen op kaart';

  @override
  String get shower => 'Douche';

  @override
  String get skiing => 'Skiën';

  @override
  String get slogan => 'Bewaar je avonturen!';

  @override
  String get slope => 'helling';

  @override
  String get someone => 'Iemand';

  @override
  String get sort => 'Sorteren';

  @override
  String get spanish => 'Spaans';

  @override
  String get speed => 'Snelheid';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statistieken';

  @override
  String get stop_drawing => 'Stop met tekenen';

  @override
  String get stop_editing => 'Stop met bewerken';

  @override
  String get strava_integration_after_date_hint =>
      'Als uw account een grote hoeveelheid activiteiten heeft, kunt u op de API-limiet van Strava botsen voorkomend dat u alle activiteiten tegelijk synchroniseert. Om dit probleem te omzeilen kunt u een \"Later\" datum hieronder instellen, zodat alleen activiteiten die na deze datum werden opgenomen worden gesynchroniseerd.';

  @override
  String get subway_stop => 'Metro toegang';

  @override
  String get summit => 'Top';

  @override
  String get summit_book => 'Bergtopboek';

  @override
  String get table => 'Tabel';

  @override
  String get tags => 'Labels';

  @override
  String get text => 'Tekst';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Aangepaste tegelsets';

  @override
  String get time => 'Time';

  @override
  String get toilets => 'Toiletten';

  @override
  String get top_speed => 'Topsnelheid';

  @override
  String get tourism => 'Toerisme';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routes',
      one: 'Route',
    );
    return '$_temp0';
  }

  @override
  String get trail_copied_successfully => 'trail copied successfully';

  @override
  String get trail_has_no_gpx => 'This trail has no GPX data.';

  @override
  String get trail_not_in_list => 'Trail is not in any list';

  @override
  String get trail_not_shared => 'Niet gedeeld met iemand';

  @override
  String get trail_saved_successfully => 'Route succesvol bewaard';

  @override
  String get trails_for_you => 'Routes voor jou';

  @override
  String get tram_stop => 'Tram halte';

  @override
  String get unchanged => 'Ongewijzigd';

  @override
  String get units => 'Eenheden';

  @override
  String get unlink => 'Ontkoppel';

  @override
  String get upload_file => 'Bestand uploaden';

  @override
  String get upload_gpx => 'GPX-bestand uploaden';

  @override
  String get upload_new_file => 'Nieuw bestand uploaden';

  @override
  String get uploaded => 'geüpload';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Gebruik heuvels';

  @override
  String get use_roads => 'Gebruik wegen';

  @override
  String get users => 'Users';

  @override
  String get username => 'Gebruikersnaam';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Weergave';

  @override
  String get viewpoint => 'Uitzichtpunt';

  @override
  String get visibilty => 'Zichtbaarheid';

  @override
  String get visibilty_status => 'Zichtbaarheid status';

  @override
  String get walking_speed => 'Wandelsnelheid';

  @override
  String get water => 'Water';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Routepunten',
      one: 'Routepunt',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Breedte';

  @override
  String get wrong_username_or_password =>
      'Onjuiste gebruikersnaam of wachtwoord';

  @override
  String get you_can => 'Jij kunt';

  @override
  String get you_have_arrived => 'You\'ve arrived';
}
