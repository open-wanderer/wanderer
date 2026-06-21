// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get biking => 'Biking';

  @override
  String get canoeing => 'Canoeing';

  @override
  String get walking => 'Walking';

  @override
  String get about => 'О';

  @override
  String get appearance => 'Appearance';

  @override
  String get account => 'Account';

  @override
  String get account_delete_confirm =>
      'Вы собираетесь удалить аккаунт. Все ваши треки также будут удалены. Продолжить?';

  @override
  String get account_privacy => 'Приватность аккаунта';

  @override
  String activity(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Активностей',
      one: 'Активность',
    );
    return '$_temp0';
  }

  @override
  String get add_bio => 'Рассказать о себе';

  @override
  String get add_entry => 'Добавить запись';

  @override
  String get add_to_list => 'Добавить в список';

  @override
  String get add_waypoint => 'Добавить путевую точку';

  @override
  String get added_trail_to => 'Трек добавлен в';

  @override
  String get added_trails_to => 'Added trails to';

  @override
  String get after => 'После';

  @override
  String get all => 'All';

  @override
  String get all_activities => 'Все активности';

  @override
  String get allow_auto_geolocate =>
      'Begin drawing a new trail from your current location';

  @override
  String get alphabetical => 'По алфавиту';

  @override
  String get already_account => 'Уже есть аккаунт?';

  @override
  String get altitude => 'Высота';

  @override
  String get amenity => 'Amenity';

  @override
  String get api_documentation => 'Документация API';

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
  String get author => 'Автор';

  @override
  String get avatar => 'Аватар';

  @override
  String get average_speed => 'Ср. скорость';

  @override
  String get avoid_bad_surfaces => 'Избегать неровные дороги';

  @override
  String get back => 'Назад';

  @override
  String get back_to_login => 'Вернуться ко входу';

  @override
  String get bakery => 'Пекарня';

  @override
  String get barrier => 'Barrier';

  @override
  String get basic_info => 'Основная информация';

  @override
  String get basque => 'Basque';

  @override
  String get before => 'До';

  @override
  String get behavior => 'Behavior';

  @override
  String get bicycle_parking => 'Велосипедная парковка';

  @override
  String get bicycle_rental => 'Прокат велосипедов';

  @override
  String get bicycle_shop => 'Веломагазин';

  @override
  String get bike_type => 'Тип велосипеда';

  @override
  String get bus_stop => 'Автобусная остановка';

  @override
  String get by => 'От';

  @override
  String get campsite => 'Campsite';

  @override
  String get can => 'можно';

  @override
  String get cancel => 'Отмена';

  @override
  String get car => 'Автомобиль:';

  @override
  String get car_motorcycle => 'Автомобиль/мотоцикл';

  @override
  String card(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Карточки',
      one: 'Карточка',
    );
    return '$_temp0';
  }

  @override
  String get categories => 'Категории';

  @override
  String get category => 'Категория';

  @override
  String get change => 'Изменить';

  @override
  String get change_email => 'Изменить email';

  @override
  String get change_password => 'Изменить пароль';

  @override
  String get changelog => 'История изменений';

  @override
  String get chinese => 'Китайский (упрощ.)';

  @override
  String get clear_all => 'Очистить всё';

  @override
  String get climbing => 'Climbing';

  @override
  String get close => 'Закрыть';

  @override
  String get collapse_trail_list => 'Collapse trail list';

  @override
  String comment(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Комментарии',
      one: 'Комментарий',
    );
    return '$_temp0';
  }

  @override
  String get completed => 'Завершено';

  @override
  String get completed_a_trail => 'завершил(ла) трек';

  @override
  String get completed_tours => 'Пройденные маршруты';

  @override
  String get completion_status => 'Статус прохождения';

  @override
  String get confirm => 'Подтвердить';

  @override
  String get confirm_deletion => 'Подтвердите удаление';

  @override
  String get confirm_publish => 'Подтвердить публикацию';

  @override
  String get confirm_share => 'Подтвердить общий доступ';

  @override
  String get connect => 'Подключить';

  @override
  String get contribute => 'Помощь в разработке';

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
  String get copy_link => 'Копировать ссылку';

  @override
  String get create_new_list => 'Создать новый список';

  @override
  String get create_waypoint => 'Создать путевую точку';

  @override
  String get creation_date => 'Дата создания';

  @override
  String get crop => 'Crop';

  @override
  String get cross => 'Циклокросс';

  @override
  String get current_password => 'Текущий пароль';

  @override
  String get cycling => 'Велосипед';

  @override
  String get cycling_speed => 'Скорость велосипеда';

  @override
  String get czech => 'Czech';

  @override
  String get danger_zone => 'Внимание!';

  @override
  String get date => 'Дата';

  @override
  String get default_category => 'Категория по умолчанию';

  @override
  String get default_location => 'Место по умолчанию';

  @override
  String get degrees => 'Градусы';

  @override
  String get delete => 'Удалить';

  @override
  String get open => 'Open';

  @override
  String get delete_account => 'Удалить аккаунт';

  @override
  String get delete_list_confirm =>
      'Удалить этот список? Треки в нём останутся доступны.';

  @override
  String get delete_summit_log_confirm =>
      'Do you really want to delete this summit log? This action cannot be undone.';

  @override
  String get delete_trail_confirm => 'Удалить этот трек? Действие необратимо.';

  @override
  String get describe_your_trail => 'Описание трека';

  @override
  String get description => 'Описание';

  @override
  String get difficult => 'Сложный';

  @override
  String get difficulty => 'Сложность';

  @override
  String get directions => 'Как добраться';

  @override
  String get display => 'Отображение';

  @override
  String get display_as => 'Отображать как';

  @override
  String get distance => 'Расстояние';

  @override
  String get documentation => 'Документация';

  @override
  String get download => 'Скачать';

  @override
  String get draw_a_route => 'Нарисовать маршрут';

  @override
  String get driving => 'На машине';

  @override
  String get duplicate => 'Дублировать';

  @override
  String get duration => 'Длительность';

  @override
  String get dutch => 'Голландский';

  @override
  String get easy => 'Лёгкий';

  @override
  String get edit => 'Редактировать';

  @override
  String get edit_entry => 'Редактировать запись';

  @override
  String get edit_list => 'Редактировать список';

  @override
  String get edit_route => 'Редактировать маршрут';

  @override
  String get edit_waypoint => 'Редактировать путевую точку';

  @override
  String get edited => 'изменено';

  @override
  String get elevation_gain => 'Набор высоты';

  @override
  String get elevation_loss => 'Сброс высоты';

  @override
  String get elevation_profile => 'Elevation Profile';

  @override
  String get email => 'Эл. почта';

  @override
  String get email_not_unique => 'This email address is already in use.';

  @override
  String get email_updated => 'Email обновлён';

  @override
  String get email_verified => 'Email подтверждён';

  @override
  String empty_activities(Object username) {
    return 'у $username ещё нет активностей';
  }

  @override
  String empty_bio(Object username) {
    return '$username ещё не рассказал(а) о себе';
  }

  @override
  String get empty_feed => 'Ваша лента пуста';

  @override
  String get empty_feed_explanation =>
      'Activities by you or people you follow will appear here';

  @override
  String empty_lists(Object username) {
    return 'у $username нет публичных списков';
  }

  @override
  String get enable_auto_routing => 'Авто-маршрутизация';

  @override
  String get english => 'Английский';

  @override
  String get entry => 'Запись';

  @override
  String get error_copying_trail => 'Error copying trail';

  @override
  String get error_creating_user => 'Ошибка создания пользователя';

  @override
  String get error_deleting_token => 'Error deleting token';

  @override
  String get error_disabling_strava_integration => 'Ошибка отключения Strava';

  @override
  String get error_during_login => 'Ошибка входа';

  @override
  String get error_during_password_reset =>
      'Не удалось отправить email сброса пароля';

  @override
  String get error_exporting_trail => 'Ошибка экспорта трека';

  @override
  String get error_generating_token => 'Error generating token';

  @override
  String get error_liking_trail => 'Error liking trail';

  @override
  String get error_logging_in_to_hammerhead => 'Error logging in to Hammerhead';

  @override
  String get error_logging_in_to_komoot => 'Ошибка входа в Komoot';

  @override
  String get error_posting_comment => 'Ошибка отправки комментария';

  @override
  String get error_printing_map => 'Ошибка печати карты';

  @override
  String get error_reading_file => 'Ошибка чтения файла';

  @override
  String get error_saving_list => 'Ошибка сохранения списка';

  @override
  String get error_saving_settings => 'Error saving settings';

  @override
  String get error_saving_trail => 'Ошибка сохранения трека';

  @override
  String error_setting_up_integration(Object provider) {
    return 'Ошибка настройки $provider';
  }

  @override
  String get error_updating_hammerhead_integration =>
      'Error updating Hammerhead integration';

  @override
  String get error_updating_komoot_integration =>
      'Error updating komoot integration';

  @override
  String get error_updating_password => 'Ошибка изменения пароля';

  @override
  String get error_updating_strava_integration => 'Ошибка обновления Strava';

  @override
  String get error_uploading_trail_to_hammerhead =>
      'Error uploading trail to Hammerhead';

  @override
  String get est_duration => 'Продолжительность';

  @override
  String get everyone_with_the_link => 'Everyone with the link';

  @override
  String get expand_trail_list => 'Expand trail list';

  @override
  String get expiration => 'Expiration';

  @override
  String get expires => 'Expires';

  @override
  String get explore => 'Изучить';

  @override
  String get explore_some_trails => 'Изучите треки';

  @override
  String get exit_navigation => 'Exit';

  @override
  String get stop_navigation_confirm =>
      'Stop navigation and return to the trail?';

  @override
  String get search_this_area => 'Search this area';

  @override
  String get export => 'Экспорт';

  @override
  String get export_all_trails => 'Экспорт всех треков';

  @override
  String get favourite_sport => 'Любимый вид спорта';

  @override
  String get features => 'Особенности';

  @override
  String get ferry => 'Паром';

  @override
  String get file_format => 'Формат файла';

  @override
  String file_too_big(Object file, Object size) {
    return 'Файл $file слишком большой (max. $size)';
  }

  @override
  String get filter_categories => 'Фильтр категорий';

  @override
  String get filter_difficulty => 'Фильтр сложности';

  @override
  String get filter_tags => 'Фильтр меток';

  @override
  String get filter_trails => 'Filter trails';

  @override
  String get finish => 'Финиш';

  @override
  String get fixed_speed => 'Крейсерская скорость';

  @override
  String get focus_map_on => 'Фокусироваться на';

  @override
  String get follow => 'Подписаться';

  @override
  String get follow_request_pending => 'Запрос обрабатывается';

  @override
  String get followers => 'Подписчики';

  @override
  String get following => 'Подписки';

  @override
  String get food => 'Food';

  @override
  String get food_drinks => 'Food & Drinks';

  @override
  String get forgot_your_password => 'Забыли пароль?';

  @override
  String get french => 'Французский';

  @override
  String get from_file => 'Из файла';

  @override
  String get from_photos => 'Из фотографии';

  @override
  String get from_url => 'По ссылке';

  @override
  String get garage => 'Гараж';

  @override
  String get gas_station => 'Gas station';

  @override
  String get generate_new_token => 'Generate new token';

  @override
  String get german => 'Немецкий';

  @override
  String get get_position_from_exif => 'Координаты из EXIF';

  @override
  String get get_started => 'Get started';

  @override
  String get grid => 'Сетка';

  @override
  String get grocery_store => 'Продуктовый магазин';

  @override
  String get hammerhead_integration_after_date_hint =>
      'If your hammerhead account is already synced with other trail databases, such as komoot or Strava, start syncing your Hammerhead data may result in duplicates. To avoid this, you can set an start date below, meaning only activities recorded after this date will be synced.';

  @override
  String get heading => 'Heading';

  @override
  String get height => 'Высота';

  @override
  String get help => 'Помощь';

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
  String get hiking => 'Пеший туризм';

  @override
  String get home => 'Главная';

  @override
  String get hotel => 'Гостиница';

  @override
  String get hungarian => 'Венгерский';

  @override
  String get hut => 'Хижина';

  @override
  String get hybrid => 'Гибрид';

  @override
  String get icon => 'Иконка';

  @override
  String get ignore_trails_before_date => 'Ignore trails before this date';

  @override
  String get imperial => 'Имперская';

  @override
  String get import => 'Импорт';

  @override
  String get import_hint => 'Перетащите GPX, FIT, KML или TCX файлы сюда...';

  @override
  String get include_description => 'Добавить описание';

  @override
  String get include_waypoints => 'Include waypoints';

  @override
  String get integration_description_hammerhead =>
      'Syncs your Hammerhead tours with wanderer in regular intervals.';

  @override
  String get integration_description_komoot =>
      'Синхронизирует ваши данные с Komoot.';

  @override
  String get integration_description_strava =>
      'Синхронизирует ваши данные со Strava.';

  @override
  String get integration_disabled => 'интеграция отключена';

  @override
  String get integration_enabled => 'интеграция включена';

  @override
  String get integration_privacy_hint_original =>
      'Imported trails will maintain the same visibility they have on the external platform. For example, if the original trail was public, it will be public in wanderer, even if trails are private by default according to your privacy settings.';

  @override
  String get integration_privacy_hint_user =>
      'The original trail\'s visibility is discarded. Instead, the local privacy settings for trails are applied to all imported trails.';

  @override
  String get integrations => 'Интеграции';

  @override
  String get invalid_date => 'Неверная дата';

  @override
  String get invalid_username => 'Некорректное имя пользователя';

  @override
  String get italian => 'Итальянский';

  @override
  String get joined => 'Зарегистрирован';

  @override
  String get keep_original => 'Keep original';

  @override
  String get keep_private => 'Keep private';

  @override
  String get language => 'Язык';

  @override
  String get last_used => 'Last used';

  @override
  String get latitude => 'Широта';

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
  String get license => 'Лицензия';

  @override
  String get like_status => 'Статус лайка';

  @override
  String get liked => 'Лайкнуто';

  @override
  String get likes => 'Лайков';

  @override
  String get limited => 'Ограничено';

  @override
  String get link_copied => 'Ссылка скопирована!';

  @override
  String get linked_lists => 'Linked lists';

  @override
  String list(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Списки',
      one: 'Список',
    );
    return '$_temp0';
  }

  @override
  String get list_not_shared => 'Нет общего доступа';

  @override
  String get list_public_warning => 'Все треки в списке станут публичными.';

  @override
  String get list_saved_successfully => 'Список сохранён';

  @override
  String get list_share_warning =>
      'Общий доступ к списку открывает доступ ко всем трекам в нём.';

  @override
  String get list_share_warning_update =>
      'Новые треки будут доступны всем, у кого есть доступ к списку.';

  @override
  String get location => 'Местоположение';

  @override
  String get locations => 'Locations';

  @override
  String get login => 'Войти';

  @override
  String get login_details => 'Данные для входа';

  @override
  String get logout => 'Выйти';

  @override
  String get longitude => 'Долгота';

  @override
  String get loop => 'Круг';

  @override
  String get make_one => 'Создайте!';

  @override
  String get make_thumbnail => 'Сделать миниатюру';

  @override
  String get map => 'Карта';

  @override
  String get map_style => 'Стиль карты';

  @override
  String get max_hiking_difficulty => 'Макс. сложность';

  @override
  String get metric => 'Метрическая';

  @override
  String get moderate => 'Средний';

  @override
  String get more => 'More';

  @override
  String get more_route_settings => 'More route settings';

  @override
  String get mountain => 'Горный';

  @override
  String get mountain_pass => 'Горный перевал';

  @override
  String must_be_at_least_n_characters_long(Object n) {
    return 'Минимум $n символов';
  }

  @override
  String must_be_at_most_n_characters_long(Object n) {
    return 'Максимум $n символов';
  }

  @override
  String get my_account => 'Мой аккаунт';

  @override
  String get my_trails => 'My trails';

  @override
  String in_distance(String distance) {
    return 'in $distance';
  }

  @override
  String n_days_ago(Object n) {
    return '$n дней назад';
  }

  @override
  String n_hours_ago(Object n) {
    return '$n часов назад';
  }

  @override
  String n_minutes_ago(Object n) {
    return '$n минут назад';
  }

  @override
  String n_months_ago(Object n) {
    return '$n месяцев назад';
  }

  @override
  String n_seconds_ago(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n секунд назад',
      zero: 'только что',
    );
    return '$_temp0';
  }

  @override
  String n_years_ago(Object n) {
    return '$n лет назад';
  }

  @override
  String get name => 'Название';

  @override
  String get navigate => 'Navigate';

  @override
  String get near => 'Рядом';

  @override
  String get never => 'Never';

  @override
  String get new_list => 'Новый список';

  @override
  String get new_password => 'Новый пароль';

  @override
  String get new_password_error => 'Ошибка обновления пароля';

  @override
  String get new_password_success => 'Пароль успешно изменен';

  @override
  String get new_password_text => 'Установить новый пароль';

  @override
  String get new_token_generated => 'New API Token generated';

  @override
  String get new_trail => 'Новый трек';

  @override
  String get no_account => 'Нет аккаунта?';

  @override
  String get no_api_tokens => 'You have no API Tokens';

  @override
  String get no_comments_so_far => 'Пока нет комментариев';

  @override
  String get no_data => 'Нет данных';

  @override
  String get no_description_for_now => 'Пока нет описания';

  @override
  String get no_gps_data_in_image => 'No GPS data in image';

  @override
  String get no_grid => 'Без сетки';

  @override
  String get no_notifications => 'Нет уведомлений';

  @override
  String get no_photos_here => 'Здесь нет фото/видео';

  @override
  String get no_preference => 'Любые';

  @override
  String get no_results => 'Ничего не найдено';

  @override
  String get no_routes_added => 'Нет добавленных маршрутов';

  @override
  String get no_waypoints_yet => 'Нет путевых точек';

  @override
  String get norwegian => 'Norwegian';

  @override
  String get not_a_valid_email_address => 'Некорректный email';

  @override
  String get not_a_valid_url => 'Некорректный URL';

  @override
  String get not_completed => 'Не завершено';

  @override
  String notification_comment_mention(Object user) {
    return '$user упомянул вас в комментарии';
  }

  @override
  String notification_list_create(Object user) {
    return '$user создал(а) новый список';
  }

  @override
  String notification_list_share(Object user) {
    return '$user предоставил(а) доступ к списку';
  }

  @override
  String get notification_new_follower => 'У вас новый подписчик';

  @override
  String notification_summit_log_create(Object trail, Object user) {
    return '$user created a summit log on your trail \"$trail\"';
  }

  @override
  String notification_summit_log_mention(Object user) {
    return '$user mentioned you in a summit log';
  }

  @override
  String notification_trail_comment(Object trail, Object user) {
    return '$user прокомментировал(а) ваш трек \"$trail\"';
  }

  @override
  String notification_trail_create(Object user) {
    return '$user создал(а) новый трек';
  }

  @override
  String notification_trail_like(Object trail, Object user) {
    return '$user лайкнул ваш трек \"$trail\"';
  }

  @override
  String notification_trail_mention(Object user) {
    return '$user упомянул вас в треке';
  }

  @override
  String notification_trail_share(Object user) {
    return '$user предоставил(а) доступ к треку';
  }

  @override
  String get notifications => 'Уведомления';

  @override
  String object_share_error(Object object) {
    return '$object должен быть публичным для совместного использования в разных экземплярах.';
  }

  @override
  String get off => 'Выкл';

  @override
  String get only_me => 'Только я';

  @override
  String get open_in_new_tab => 'Открыть в новой вкладке';

  @override
  String get or => 'или';

  @override
  String get orientation => 'Ориентация';

  @override
  String get paper_size => 'Размер бумаги';

  @override
  String get paragraph => 'Paragraph';

  @override
  String get parking => 'Парковка';

  @override
  String get pause => 'Pause';

  @override
  String get password => 'Пароль';

  @override
  String get password_confirm => 'Подтвердите пароль';

  @override
  String get password_reset_sent => 'Письмо для сброса пароля отправлено';

  @override
  String get password_reset_text =>
      'Мы отправим ссылку для сброса на ваш email.';

  @override
  String get password_updated => 'Пароль обновлён';

  @override
  String get passwords_must_match => 'Пароли не совпадают';

  @override
  String get photos => 'Фото и Видео';

  @override
  String get pick_a_trail => 'Выберите трек';

  @override
  String get planned_a_trail => 'запланировал(а) трек';

  @override
  String get planned_tours => 'Запланированные маршруты';

  @override
  String get pois => 'POIs';

  @override
  String get polish => 'Польский';

  @override
  String get portuguese => 'Португальский';

  @override
  String get print => 'Печать';

  @override
  String get privacy => 'Приватность';

  @override
  String get private => 'Зкрытый';

  @override
  String get profile => 'Профиль';

  @override
  String get public => 'Открытый';

  @override
  String get public_access => 'Общий доступ';

  @override
  String get public_share_everyone =>
      'Everyone on the internet with the link can see this trail';

  @override
  String get public_share_limited =>
      'Only people with access can open the link';

  @override
  String get public_transport => 'Общественный транспорт';

  @override
  String get radius => 'Радиус';

  @override
  String get railway_station => 'Железнодорожная станция';

  @override
  String get reached_end_of_trail => 'You\'ve reached the end of the trail.';

  @override
  String get read_more => 'Подробнее';

  @override
  String get ready_to_join => 'Ready to join';

  @override
  String get recalculate_elevation_data => 'Recalculate elevation data';

  @override
  String get recalculating_elevation_data_hint =>
      'Recalculating elevation data will erase the existing elevation data, if any, and replace it with data from Valhalla.';

  @override
  String get register => 'Регистрация';

  @override
  String get remote_users_cannot_edit => 'Remote users cannot edit';

  @override
  String get removed_trail_from => 'Трек удалён из';

  @override
  String get removed_trails_from => 'Removed trails from';

  @override
  String get required => 'Обязательно';

  @override
  String get reset => 'Reset';

  @override
  String get reset_password => 'Сбросить пароль';

  @override
  String get resume => 'Resume';

  @override
  String get reverse_direction => 'Reverse direction';

  @override
  String get road => 'Шоссе';

  @override
  String route(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Маршрутов',
      one: 'Маршрут',
    );
    return '$_temp0';
  }

  @override
  String get route_point => 'Точка маршрута';

  @override
  String get russian => 'Русский';

  @override
  String get save => 'Сохранить';

  @override
  String get save_list => 'Сохранить список';

  @override
  String get save_trail => 'Сохранить трек';

  @override
  String get save_your_trail_first => 'Сначала сохраните трек';

  @override
  String get search => 'Search';

  @override
  String get search_cities => 'Поиск по городам';

  @override
  String get search_for_trails_places => 'Поиск треков, списков, мест';

  @override
  String get search_list => 'Search list';

  @override
  String get search_places => 'Поиск мест';

  @override
  String get search_trails => 'Поиск треков';

  @override
  String get select_date => 'Select date';

  @override
  String get select_list => 'Выбрать список';

  @override
  String get selected => 'selected';

  @override
  String get send_to => 'Send to...';

  @override
  String get set_private => 'Set private';

  @override
  String get set_public => 'Set public';

  @override
  String get settings => 'Настройки';

  @override
  String get settings_notification_comment_mention =>
      'Кто-то упомянул вас в комментариях';

  @override
  String get settings_notification_list_share =>
      'Вам предоставили доступ к списку';

  @override
  String get settings_notification_new_follower => 'У вас новый подписчик';

  @override
  String get settings_notification_summit_log_create =>
      'Someone created a summit log on your trail';

  @override
  String get settings_notification_summit_log_mention =>
      'Someone mentioned you in a summit log';

  @override
  String get settings_notification_trail_comment =>
      'Кто-то прокомментировал ваш трек';

  @override
  String get settings_notification_trail_like => 'Кто-то лайкнул ваш трек';

  @override
  String get settings_notification_trail_mention =>
      'Кто-то упомянул вас в треке';

  @override
  String get settings_notification_trail_share =>
      'Вам предоставили доступ к треку';

  @override
  String get settings_privacy_account_private =>
      'Только вы видите свой профиль. Не отображается при поиске. Другие пользователи не могут подписаться на вас или поделиться треками. Вы по-прежнему можете публиковать маршруты или списки.';

  @override
  String get settings_privacy_account_public =>
      'Все видят ваш профиль. Отображается при поиске. Другие пользователи могут подписаться на вас и поделиться треками.';

  @override
  String get settings_privacy_lists_private =>
      'Списки по умолчанию закрытые. Их видите только вы. Настройку можно изменить для каждого списка.';

  @override
  String get settings_privacy_lists_public =>
      'Списки по умолчанию открытые. Их видят все. Настройку можно изменить для каждого списка.';

  @override
  String get settings_privacy_trails_private =>
      'Треки по умолчанию закрытые. Их видите только вы. Настройку можно изменить для каждого трека.';

  @override
  String get settings_privacy_trails_public =>
      'Треки по умолчанию открытые. Их видят все. Настройку можно изменить для каждого трека.';

  @override
  String get settings_saved => 'Настройки сохранены';

  @override
  String get share => 'Поделиться';

  @override
  String get share_profile => 'Поделиться профилем';

  @override
  String get share_this_list => 'Открыть доступ к списку';

  @override
  String get share_this_trail => 'Открыть доступ к треку';

  @override
  String get shared => 'Общий доступ';

  @override
  String get shared_by => 'Доступ от';

  @override
  String get shared_with => 'Доступно для';

  @override
  String get shelter => 'Укрытие';

  @override
  String get shortest => 'кратчайший';

  @override
  String get show_in_overview => 'Быстрый просмотр';

  @override
  String get show_less => 'Show less';

  @override
  String get show_on_map => 'Показать на карте';

  @override
  String get shower => 'Душ';

  @override
  String get skiing => 'Skiing';

  @override
  String get slogan => 'Сохраняйте ваши приключения!';

  @override
  String get slope => 'Уклон';

  @override
  String get someone => 'Кто-то';

  @override
  String get sort => 'Сортировка';

  @override
  String get spanish => 'Испанский';

  @override
  String get speed => 'Скорость';

  @override
  String get start => 'Старт';

  @override
  String get statistics => 'Статистика';

  @override
  String get stop_drawing => 'Закончить рисование';

  @override
  String get stop_editing => 'Закончить редактирование';

  @override
  String get strava_integration_after_date_hint =>
      'If your account has a large amount of acitivities you may run into Strava\'s API rate limit preventing you from syncing all activities at once. To mitigate this issue you can set an \"After\" date below so that only activities that were recorded after this date are synced.';

  @override
  String get subway_stop => 'Вход в метро';

  @override
  String get summit => 'Вершина';

  @override
  String get summit_book => 'История поездок';

  @override
  String get table => 'Таблица';

  @override
  String get tags => 'Метки';

  @override
  String get text => 'Текст';

  @override
  String get theme_dark => 'Dark';

  @override
  String get theme_light => 'Light';

  @override
  String get theme_system => 'Follow system';

  @override
  String get tilesets => 'Пользовательские тайлы';

  @override
  String get time => 'Time';

  @override
  String get toilets => 'Туалеты';

  @override
  String get top_speed => 'Макс. скорость';

  @override
  String get tourism => 'Туризм';

  @override
  String trail(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Треки',
      one: 'Трек',
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
  String get trail_not_shared => 'Нет общего доступа';

  @override
  String get trail_saved_successfully => 'Трек сохранён';

  @override
  String get trails_for_you => 'Треки для вас';

  @override
  String get tram_stop => 'Трамвайная остановка';

  @override
  String get unchanged => 'не изменено';

  @override
  String get units => 'Единицы измерения';

  @override
  String get unlink => 'Unlink';

  @override
  String get upload_file => 'Загрузить файл';

  @override
  String get upload_gpx => 'Загрузить GPX';

  @override
  String get upload_new_file => 'Загрузить новый файл';

  @override
  String get uploaded => 'загружено';

  @override
  String get uploaded_trail_to_hammerhead =>
      'Successfully uploaded trail to Hammerhead';

  @override
  String get use_hills => 'Use hills';

  @override
  String get use_roads => 'Использование дорог';

  @override
  String get users => 'Users';

  @override
  String get username => 'Никнейм';

  @override
  String get username_not_unique =>
      'This username is already taken. Please try another.';

  @override
  String get view => 'Просмотр';

  @override
  String get viewpoint => 'Viewpoint';

  @override
  String get visibilty => 'Visibility';

  @override
  String get visibilty_status => 'Статус видимости';

  @override
  String get walking_speed => 'Пешая скорость';

  @override
  String get water => 'Вода';

  @override
  String get web => 'Web';

  @override
  String waypoints(num n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Путевые точки',
      one: 'Путевая точка',
    );
    return '$_temp0';
  }

  @override
  String get welcome_to => 'Welcome to';

  @override
  String get width => 'Ширина';

  @override
  String get wrong_username_or_password => 'Неверный логин или пароль';

  @override
  String get you_can => 'Вы можете';

  @override
  String get you_have_arrived => 'You\'ve arrived';
}
