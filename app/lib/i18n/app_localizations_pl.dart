// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Polish (`pl`).
class AppLocalizationsPl extends AppLocalizations {
  AppLocalizationsPl([String locale = 'pl']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'Na temat';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Za chwilę usuniesz swoje konto. Wszystkie twoje szlaki zostaną usunięte. Czy chcesz kontynuować?';

  @override
  String get account_privacy => 'Prywatność konta';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Aktywności',
      one: 'Aktywność',
      many: 'Aktywności',
      few: 'Aktywności',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Dodaj biogram';

  @override
  String get add_entry => 'Dodaj Pozycję';

  @override
  String get add_to_list => 'Dodaj do listy';

  @override
  String get add_waypoint => 'Dodaj Punkt';

  @override
  String get added_trail_to => 'Dodaj szlak do';

  @override
  String get added_trails_to => 'Dodaj szlaki do';

  @override
  String get after => 'Po';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Wszystkie aktywności';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'Alfabetyczne';

  @override
  String get already_account => 'Czy masz już konto?';

  @override
  String get altitude => 'Wysokość';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'Dokumentacja API';

  @override
  String get api_tokens => 'API Tokens';

  @override
  String get api_tokens_hint =>
      'API Tokens can be used to grant 3rd party applications access to your wanderer account.';

  @override
  String get apply_user_settings => 'Apply user settings';

  @override
  String get attraction => 'Attraction';

  @override
  String get author => 'Autor';

  @override
  String get avatar => 'Awatar';

  @override
  String get average_speed => 'Śr. prędkość';

  @override
  String get avoid_bad_surfaces => 'Unikaj złych nawierzchni';

  @override
  String get back => 'Wstecz';

  @override
  String get back_to_login => 'Powrót do logowania';

  @override
  String get bakery => 'Bakery';

  @override
  String get barrier => 'Barrier';

  @override
  String get basic_info => 'Podstawowe informacje';

  @override
  String get basque => 'Basque';

  @override
  String get before => 'Przed';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Bicycle Parking';

  @override
  String get bicycle_rental => 'Bicycle Rental';

  @override
  String get bicycle_shop => 'Bicycle Shop';

  @override
  String get bike_type => 'Typ roweru';

  @override
  String get bus_stop => 'Bus stop';

  @override
  String get by => 'przez';

  @override
  String get campsite => 'Campsite';

  @override
  String get can => 'może';

  @override
  String get cancel => 'Anuluj';

  @override
  String get car => 'Samochód';

  @override
  String get car_motorcycle => 'Car/Motorcycle';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Karty',
      one: 'Karta',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Kategorie';

  @override
  String get category => 'Kategoria';

  @override
  String get change => 'Zmień';

  @override
  String get change_email => 'Zmień adres e-mail';

  @override
  String get change_password => 'Zmień hasło';

  @override
  String get changelog => 'Dziennik zmian';

  @override
  String get chinese => 'Chiński (uproszczony)';

  @override
  String get clear_all => 'Wyczyść wszystko';

  @override
  String get climbing => 'Climbing';

  @override
  String get close => 'Zamknij';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Komentarze',
      one: 'Komentarz',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Zakończono';

  @override
  String get completed_a_trail => 'ukończono szlak';

  @override
  String get completed_tours => 'Przebyte trasy';

  @override
  String get completion_status => 'Stan ukończenia';

  @override
  String get confirm => 'Potwierdź';

  @override
  String get confirm_deletion => 'Potwierdź usunięcie';

  @override
  String get confirm_publish => 'Potwierdź publikację';

  @override
  String get confirm_share => 'Potwierdź udostępnienie';

  @override
  String get connect => 'Połącz';

  @override
  String get contribute => 'Kontrybuuj';

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
  String get location_unavailable =>
      'Unable to determine your location. Please try again.';

  @override
  String get copy_link => 'Kopiuj link';

  @override
  String get create_new_list => 'Stwórz nową listę';

  @override
  String get create_waypoint => 'Utwórz punkt trasy';

  @override
  String get creation_date => 'Data dodania';

  @override
  String get crop => 'Crop';

  @override
  String get cross => 'Krzyż';

  @override
  String get current_password => 'Obecne hasło';

  @override
  String get cycling => 'Rower';

  @override
  String get cycling_speed => 'Prędkość jazdy na rowerze';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Strefa niebezpieczna';

  @override
  String get date => 'Data';

  @override
  String get default_category => 'Domyślna kategoria';

  @override
  String get default_location => 'Domyślna Lokalizacja';

  @override
  String get degrees => 'Stopnie';

  @override
  String get delete => 'Usuń';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Usuń Konto';

  @override
  String get delete_list_confirm =>
      'Czy na pewno chcesz usunąć tę listę? Szlaki na liście zostaną zachowane.';

  @override
  String get delete_summit_log_confirm =>
      'Czy na pewno chcesz usunąć ten dziennik szczytu? Tej akcji nie można cofnąć.';

  @override
  String get delete_trail_confirm =>
      'Czy na pewno chcesz usunąć ten szlak? Ta akcja jest nieodwracalna.';

  @override
  String get describe_your_trail => 'Opisz swój szlak';

  @override
  String get description => 'Opis';

  @override
  String get difficult => 'Trudna';

  @override
  String get difficulty => 'Trudność';

  @override
  String get directions => 'Kierunki';

  @override
  String get display => 'Wyświetlanie';

  @override
  String get display_as => 'Wyświetl jako';

  @override
  String get distance => 'Dystans';

  @override
  String get documentation => 'Dokumentacja';

  @override
  String get download => 'Pobierz';

  @override
  String get draw_a_route => 'Narysuj trasę';

  @override
  String get driving => 'Samochód';

  @override
  String get duplicate => 'Duplikuj';

  @override
  String get duration => 'Czas trwania';

  @override
  String get dutch => 'Niderlandzki';

  @override
  String get easy => 'Łatwy';

  @override
  String get edit => 'Edytuj';

  @override
  String get edit_entry => 'Edytuj pozycję';

  @override
  String get edit_list => 'Edytuj Listę';

  @override
  String get edit_route => 'Edytuj trasę';

  @override
  String get edit_waypoint => 'Edytuj Punkt';

  @override
  String get edited => 'edytowany';

  @override
  String get elevation_gain => 'Przyrost wysokości';

  @override
  String get elevation_loss => 'Strata wysokości';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'E-mail';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Email zaktualizowany';

  @override
  String get email_verified => 'Adres e-mail został zweryfikowany';

  @override
  String empty_activities(Object username) {
    return '$username nie ma jeszcze aktywności';
  }

  @override
  String empty_bio(Object username) {
    return '$username jeszcze nie dodał biogramu';
  }

  @override
  String get empty_feed => 'Your feed is empty';

  @override
  String get empty_feed_explanation =>
      'Activities by you or people you follow will appear here';

  @override
  String empty_lists(Object username) {
    return '$username nie ma publicznych list';
  }

  @override
  String get enable_auto_routing => 'Włącz auto-trasowanie';

  @override
  String get english => 'Angielski';

  @override
  String get entry => 'Pozycja';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Błąd tworzenia użytkownika';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration =>
      'Błąd przy wyłączaniu integracji strava';

  @override
  String get error_during_login => 'Błąd podczas logowania';

  @override
  String get error_during_password_reset =>
      'Nie udało się wysłać e-maila z resetowaniem hasła';

  @override
  String get error_exporting_trail => 'Błąd podczas eksportowania szlaku';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Error liking trail';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Błąd zapisu do komoot';

  @override
  String get error_posting_comment => 'Błąd wysyłania komentarza';

  @override
  String get error_printing_map => 'Błąd podczas drukowania mapy';

  @override
  String get error_reading_file => 'Błąd wczytywania pliku';

  @override
  String get error_saving_list => 'Błąd przy zapisywaniu listy';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Błąd podczas zapisywania szlaku';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Error setting up strava integration';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Błąd podczas aktualizacji hasła';

  @override
  String get error_updating_strava_integration =>
      'Błąd aktualizacji integracji kamoot';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Szacowany czas';

  @override
  String get everyone_with_the_link => 'Everyone with the link';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Eksploruj';

  @override
  String get explore_some_trails => 'Eksploruj różne szlaki';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get stop_recording => 'Zatrzymaj nagrywanie';

  @override
  String get stop_recording_confirm => 'Zatrzymać nagrywanie?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Eksportuj';

  @override
  String get export_all_trails => 'Eksportuj wszystkie szlaki';

  @override
  String get favourite_sport => 'Ulubiony sport';

  @override
  String get features => 'Funkcje';

  @override
  String get ferry => 'Ferry';

  @override
  String get file_format => 'Format pliku';

  @override
  String file_too_big(Object file, Object size) {
    return 'Plik $file jest za duży (maks. $size)';
  }

  @override
  String get filter_categories => 'Filtruj kategorie';

  @override
  String get filter_difficulty => 'Filtruj poziom trudności';

  @override
  String get filter_tags => 'Filtruj tagi';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Zakończ';

  @override
  String get finish_disabled_hint =>
      'Add at least 2 anchors to finish your route.';

  @override
  String get fixed_speed => 'Stała prędkość';

  @override
  String get focus_map_on => 'Skoncentruj mapę na';

  @override
  String get follow => 'Obserwuj';

  @override
  String get follow_request_pending => 'Żądanie w toku';

  @override
  String get followers => 'Obserwujący';

  @override
  String get following => 'Obserwowane';

  @override
  String get food => 'Food';

  @override
  String get food_drinks => 'Food & Drinks';

  @override
  String get forgot_your_password => 'Zapomniałeś hasła?';

  @override
  String get french => 'Francuski';

  @override
  String get from_file => 'Z pliku';

  @override
  String get from_photos => 'Ze zdjęć';

  @override
  String get from_url => 'Z URL';

  @override
  String get garage => 'Garage';

  @override
  String get gas_station => 'Gas station';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Niemiecki';

  @override
  String get get_position_from_exif => 'Odczytaj współrzędne z danych EXIF';

  @override
  String get get_started => 'Get started';

  @override
  String get grid => 'Siatka';

  @override
  String get grocery_store => 'Grocery store';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Heading';

  @override
  String get height => 'Wysokość';

  @override
  String get help => 'Pomoc';

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
  String get hiking => 'Wędrówka';

  @override
  String get home => 'Strona główna';

  @override
  String get hotel => 'Hotel';

  @override
  String get hungarian => 'Język węgierski';

  @override
  String get hut => 'Hut';

  @override
  String get hybrid => 'Hybrydowy';

  @override
  String get icon => 'Ikona';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Anglosaskie';

  @override
  String get import => 'Importuj';

  @override
  String get import_hint =>
      'Wybierz lub przeciągnij tutaj plik GPX, FIT, KML lub TCX...';

  @override
  String get include_description => 'Dołącz opis';

  @override
  String get include_waypoints => 'Uwzględnij punkty trasy';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Synchronizuje trasy kamoot z wanderer w równych odstępach.';

  @override
  String get integration_description_strava =>
      'Synchronizuje trasy i aktywność z wanderer w równych odstępach.';

  @override
  String get integration_disabled => 'integracja wyłączona';

  @override
  String get integration_enabled => 'integracja włączona';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Integracje';

  @override
  String get invalid_date => 'Nieprawidłowa data';

  @override
  String get invalid_username => 'Błędna nazwa użytkownika';

  @override
  String get italian => 'Włoski';

  @override
  String get joined => 'Dołączono';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Język';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Szerokość';

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
  String get license => 'Licencja';

  @override
  String get like_status => 'Status polubień';

  @override
  String get liked => 'Polubiony';

  @override
  String get likes => 'Polubienia';

  @override
  String get limited => 'Limited';

  @override
  String get link_copied => 'Link skopiowany!';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Listy',
      one: 'Lista',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Nikomu nie udostępniony';

  @override
  String get list_public_warning =>
      'Wszystkie szlaki na tej liście staną się publiczne.';

  @override
  String get list_saved_successfully => 'Lista poprawnie zapisana';

  @override
  String get list_share_warning =>
      'Udostępnianie listy automatycznie udostępnia wszystkie szlaki w niej zawarte.';

  @override
  String get list_share_warning_update =>
      'Dodane szlaki będą udostępnione każdemu, kto ma dostęp do tej listy.';

  @override
  String get location => 'Lokalizacja';

  @override
  String get locations => 'Locations';

  @override
  String get center_on_my_location => 'Center on my location';

  @override
  String get location_tracking_notification_title => 'Wanderer';

  @override
  String get location_tracking_notification_text => 'Recording your trail';

  @override
  String get login => 'Zaloguj się';

  @override
  String get login_details => 'Dane logowania';

  @override
  String get logout => 'Wyloguj';

  @override
  String get longitude => 'Wysokość';

  @override
  String get loop => 'Pętla';

  @override
  String get make_one => 'Stwórz ją!';

  @override
  String get make_thumbnail => 'Zrób miniaturkę';

  @override
  String get map => 'Mapa';

  @override
  String get map_style => 'Map style';

  @override
  String get max_hiking_difficulty => 'Maksymalny poziom trudności wędrówki';

  @override
  String get metric => 'Metryczne';

  @override
  String get moderate => 'Średni';

  @override
  String get more => 'More';

  @override
  String get more_route_settings => 'More route settings';

  @override
  String get mountain => 'Góra';

  @override
  String get mountain_pass => 'Mountain pass';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Długość musi wynosić przynajmniej $n znaków';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Musi mieć długość co najwyżej $n znaków';
  }

  @override
  String get my_account => 'Moje konto';

  @override
  String get my_trails => 'My trails';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n dni temu';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n godzin temu';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n minut temu';
  }

  @override
  String n_months_ago(Object n) {
    return '$n miesięcy temu';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n sekund(y) temu',
      zero: 'teraz',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n lat temu';
  }

  @override
  String get name => 'Nazwa';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Blisko';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Nowa Lista';

  @override
  String get new_password => 'Nowe hasło';

  @override
  String get new_password_error => 'Błąd ustawiania nowego hasła';

  @override
  String get new_password_success => 'Nowe hasło zostało ustawione';

  @override
  String get new_password_text => 'Ustaw nowe hasło';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Nowy szlak';

  @override
  String get trail_source_planner => 'Open trail planner';

  @override
  String get trail_source_record => 'Record trail';

  @override
  String get trail_source_import => 'Import file';

  @override
  String get trail_source_import_error => 'Could not import file';

  @override
  String get travel_profile_hike => 'Hike';

  @override
  String get travel_profile_hike_description =>
      'Plan a route for walking or hiking.';

  @override
  String get travel_profile_bike => 'Bike';

  @override
  String get travel_profile_bike_description => 'Plan a route for cycling.';

  @override
  String get coming_soon => 'Coming soon';

  @override
  String get no_account => 'Nie masz konta?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Nie ma jeszcze komentarzy';

  @override
  String get no_data => 'Brak danych';

  @override
  String get no_description_for_now => 'Nie ma jeszcze opisu';

  @override
  String get no_gps_data_in_image => 'No GPS data in image';

  @override
  String get no_grid => 'Brak Siatki';

  @override
  String get no_notifications => 'Brak powiadomień';

  @override
  String get no_photos_here => 'Nie ma tu zdjęć';

  @override
  String get no_preference => 'Brak preferencji';

  @override
  String get no_results => 'Brak wyników';

  @override
  String get no_routes_added => 'Brak dodanych tras';

  @override
  String get no_waypoints_yet => 'Jeszcze nie ma punktów trasy';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Nieprawidłowy adres email';

  @override
  String get not_a_valid_url => 'Wadliwy URL';

  @override
  String get not_completed => 'Nie dokończono';

  @override
  String notification_comment_mention(Object user) {
    return '$user wspomniał o tobie w komentarzu';
  }

  @override
  String notification_list_create(Object user) {
    return '$user utworzył(a) nową listę';
  }

  @override
  String notification_list_share(Object user) {
    return '$user udostępnił(a) tobie listę';
  }

  @override
  String get notification_new_follower => 'Masz nowego obserwującego';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user utworzył dziennik szczytu na twoim szlaku \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user wspomniał o tobie w dzienniku szczytu';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user skomentował(a) twój szlak \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user utworzył nowy szlak';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user polubił twój szlak \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user wspomniał o tobie w szlaku';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user udostępnił(a) tobie szlak';
  }

  @override
  String get notifications => 'Powiadomienia';

  @override
  String object_share_error(Object object) {
    return '$object musi być publiczny aby być udostępniany między instancjami.';
  }

  @override
  String get off => 'Brak';

  @override
  String get only_me => 'Tylko ja';

  @override
  String get open_in_new_tab => 'Otwórz w nowej zakładce';

  @override
  String get or => 'lub';

  @override
  String get orientation => 'Orientacja';

  @override
  String get paper_size => 'Rozmiar papieru';

  @override
  String get paragraph => 'Paragraph';

  @override
  String get parking => 'Parking';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Hasło';

  @override
  String get password_confirm => 'Potwierdź hasło';

  @override
  String get password_reset_sent =>
      'E-mail z resetowaniem hasła został wysłany';

  @override
  String get password_reset_text =>
      'Wyślemy link do resetowania hasła na twój adres e-mail.';

  @override
  String get password_updated => 'Hasło zaktualizowane';

  @override
  String get passwords_must_match => 'Hasła muszą być takie same';

  @override
  String get photos => 'Zdjęcia';

  @override
  String photos_skipped_no_gps(num count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count photos skipped — no GPS data',
      one: '1 photo skipped — no GPS data',
    );
    return '$_temp0';
  }

  @override
  String get pick_a_trail => 'Wybierz szlak';

  @override
  String get planned_a_trail => 'zaplanowano szlak';

  @override
  String get planned_tours => 'Planowane trasy';

  @override
  String get pois => 'POIs';

  @override
  String get polish => 'Polski';

  @override
  String get portuguese => 'Portugalski';

  @override
  String get print => 'Drukuj';

  @override
  String get privacy => 'Prywatność';

  @override
  String get private => 'Prywatne';

  @override
  String get profile => 'Profil';

  @override
  String get public => 'Publiczny';

  @override
  String get public_access => 'Public access';

  @override
  String get public_share_everyone =>
      'Everyone on the internet with the link can see this trail';

  @override
  String get public_share_limited =>
      'Only people with access can open the link';

  @override
  String get public_transport => 'Public transport';

  @override
  String get radius => 'Promień';

  @override
  String get railway_station => 'Railway station';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Czytaj dalej';

  @override
  String get ready_to_join => 'Ready to join';

  @override
  String get recalculate_elevation_data => 'Recalculate elevation data';

  @override
  String get recalculating_elevation_data_hint =>
      'Recalculating elevation data will erase the existing elevation data, if any, and replace it with data from Valhalla.';

  @override
  String get register => 'Zarejestruj';

  @override
  String get remote_users_cannot_edit => 'Remote users cannot edit';

  @override
  String get removed_trail_from => 'Usunięto szlak z';

  @override
  String get removed_trails_from => 'Usunięto szlaki z';

  @override
  String get required => 'Wymagane';

  @override
  String get reorder_photos_hint => 'Long-press and drag to reorder photos.';

  @override
  String get reset => 'Reset';

  @override
  String get reset_password => 'Resetuj hasło';

  @override
  String get resume => 'Resume';

  @override
  String resume_navigation_prompt(String trail) {
    return 'Resume navigation on $trail?';
  }

  @override
  String get resume_recording_prompt => 'Wznowić nagrywanie?';

  @override
  String get reverse_direction => 'Reverse direction';

  @override
  String get road => 'Droga';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Trasy',
      one: 'Trasa',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Punkt trasy';

  @override
  String get russian => 'Russian';

  @override
  String get save => 'Zapisz';

  @override
  String get save_list => 'Zapisz listę';

  @override
  String get save_track => 'Save track';

  @override
  String get save_trail => 'Zapisz szlak';

  @override
  String get save_your_trail_first => 'Najpierw zapisz swój szlak';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Szukaj miasta';

  @override
  String get search_for_trails_places => 'Szukaj szlaków lub miejsc';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Szukaj miejsc';

  @override
  String get search_trails => 'Szukaj szlaków';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Wybierz Listę';

  @override
  String get selected => 'selected';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Ustawienia';

  @override
  String get settings_notification_comment_mention =>
      'Ktoś wspomniał o tobie w komentarzu';

  @override
  String get settings_notification_list_share => 'Ktoś udostępnił tobie listę';

  @override
  String get settings_notification_new_follower => 'Masz nowego obserwującego';

  @override
  String get settings_notification_summit_log_create =>
      'Ktoś utworzył dziennik szczytu na twoim szlaku';

  @override
  String get settings_notification_summit_log_mention =>
      'Ktoś wspomniał o tobie w dzienniku szczytu';

  @override
  String get settings_notification_trail_comment =>
      'Ktoś skomentował twój szlak';

  @override
  String get settings_notification_trail_like => 'Ktoś polubił twój szlak';

  @override
  String get settings_notification_trail_mention =>
      'Ktoś wspomniał o tobie w szlaku';

  @override
  String get settings_notification_trail_share => 'Ktoś udostępnił tobie szlak';

  @override
  String get settings_privacy_account_private =>
      'Tylko ty możesz zobaczyć swój profil. Nie będziesz się pojawiać w wynikach wyszukiwania. Inni użytkownicy nie mogą cię obserwować ani udostępniać ci szlaków. Wciąż możesz publikować szlaki i listy.';

  @override
  String get settings_privacy_account_public =>
      'Wszyscy mogą zobaczyć twój profil. Będziesz się pojawiać w wynikach wyszukiwania. Inni użytkownicy mogą cię obserwować i udostępniać ci szlaki.';

  @override
  String get settings_privacy_lists_private =>
      'Domyślnie, twoje listy są prywatne. Nikt poza tobą nie będzie mógł ich zobaczyć. Zawsze możesz zmieniać to ustawienie, osobno dla każdej listy.';

  @override
  String get settings_privacy_lists_public =>
      'Domyślnie, twoje listy są publiczne. Każdy będzie mógł je zobaczyć. Zawsze możesz zmieniać to ustawienie, osobno dla każdej listy.';

  @override
  String get settings_privacy_trails_private =>
      'Domyślnie, twoje szlaki są prywatne. Nikt poza tobą nie będzie mógł ich zobaczyć. Zawsze możesz zmieniać to ustawienie, osobno dla każdego szlaku.';

  @override
  String get settings_privacy_trails_public =>
      'Domyślnie, twoje szlaki są publiczne. Każdy będzie mógł je zobaczyć. Zawsze możesz zmieniać to ustawienie, osobno dla każdego szlaku.';

  @override
  String get settings_saved => 'Zapisano ustawienia';

  @override
  String get share => 'Udostępnij';

  @override
  String get share_profile => 'Udostępnij profil';

  @override
  String get share_this_list => 'Udostępnij tę listę';

  @override
  String get share_this_trail => 'Udostępnij ten szlak';

  @override
  String get shared => 'Udostępniono';

  @override
  String get shared_by => 'Udostępniony przez';

  @override
  String get shared_with => 'Udostępniony dla';

  @override
  String get shelter => 'Shelter';

  @override
  String get shortest => 'shortest';

  @override
  String get show_in_overview => 'Pokaż w przeglądzie';

  @override
  String get show_less => 'Show less';

  @override
  String get show_on_map => 'Pokaż na mapie';

  @override
  String get shower => 'Shower';

  @override
  String get skiing => 'Skiing';

  @override
  String get slogan => 'Zapisz swoją wyprawę!';

  @override
  String get slope => 'Nachylenie';

  @override
  String get someone => 'Ktoś';

  @override
  String get sort => 'Sortowanie';

  @override
  String get spanish => 'hiszpański';

  @override
  String get speed => 'Prędkość';

  @override
  String get start => 'Start';

  @override
  String get statistics => 'Statystyki';

  @override
  String get stop_drawing => 'Przestań rysować';

  @override
  String get stop_editing => 'Zakończ edycję';

  @override
  String get strava_integration_after_date_hint =>
      'If your account has a large amount of acitivities you may run into Strava\'s API rate limit preventing you from syncing all activities at once. To mitigate this issue you can set an \"After\" date below so that only activities that were recorded after this date are synced.';

  @override
  String get subcategories => 'Podkategorie';

  @override
  String get subway_stop => 'Subway entrance';

  @override
  String get summit => 'Summit';

  @override
  String get summit_book => 'Logbook';

  @override
  String get table => 'Tabela';

  @override
  String get tags => 'Tagi';

  @override
  String get text => 'Tekst';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Niestandardowe zestawy płytek';

  @override
  String get time => 'Time';

  @override
  String get time_in_motion => 'Time in Motion';

  @override
  String get toilets => 'Toilets';

  @override
  String get top_speed => 'Maksymalna prędkość';

  @override
  String get tourism => 'Tourism';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Szlaki',
      one: 'Szlak',
      many: 'Szlaków',
      few: 'Szlaki',
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
  String get trail_not_shared => 'Szlak nie udostępniony';

  @override
  String get trail_saved_successfully => 'Szlak pomyślnie zapisany';

  @override
  String get some_waypoints_failed_to_save =>
      'Trail saved, but some waypoints failed to save';

  @override
  String get trails_for_you => 'Szlaki dla ciebie';

  @override
  String get tram_stop => 'Tram stop';

  @override
  String get unchanged => 'bez zmian';

  @override
  String get units => 'Jednostki';

  @override
  String get unlink => 'Unlink';

  @override
  String get upload_file => 'Przesyłanie pliku';

  @override
  String get upload_gpx => 'Importuj GPX';

  @override
  String get upload_new_file => 'Prześlij nowy plik';

  @override
  String get uploaded => 'wgrany';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Use hills';

  @override
  String get use_roads => 'Użyj dróg';

  @override
  String get users => 'Users';

  @override
  String get username => 'Nazwa Użytkownika';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Widok';

  @override
  String get viewpoint => 'Viewpoint';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => 'Status widoczności';

  @override
  String get walking_speed => 'Walking speed';

  @override
  String get water => 'Water';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Punktów',
      one: 'Punkt',
      many: 'Punktów',
      few: 'Punkty',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Szerokość';

  @override
  String get wrong_username_or_password => 'Zła nazwa użytkownika lub hasło';

  @override
  String get you_can => 'Możesz';

  @override
  String get you_have_arrived => 'You\'ve arrived';

  @override
  String get settings_categories_confirm_disable_title => 'Hide this category?';

  @override
  String get settings_categories_confirm_disable_subcategory_title =>
      'Hide this subcategory?';

  @override
  String settings_categories_confirm_disable_body(int count) {
    return '$count of your trails use this category. They will stay published but this filter will be hidden.';
  }

  @override
  String get settings_categories_confirm_view_trails => 'View trails';

  @override
  String get settings_categories_confirm_disable_confirm => 'Disable anyway';

  @override
  String get settings_categories_empty_title => 'No subcategories';

  @override
  String get settings_categories_empty_body =>
      'This category has no subcategories to configure.';

  @override
  String get settings_categories_reorder_hint =>
      'Categories control which trail types you see and in what order. Turn one off to hide it as a filter — your trails stay published, they just won\'t appear under that category. Tap a category to manage its subcategories individually.\n\nTo change the order, press and hold a row, then drag it to a new position. The order you set here is reflected everywhere categories are shown.';
}
